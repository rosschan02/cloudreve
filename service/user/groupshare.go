package user

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

const (
	GroupShareStatusOwner    = "owner"
	GroupShareStatusMember   = "member"
	GroupShareStatusPending  = "pending"
	GroupShareStatusJoinable = "joinable"
)

type (
	// GroupShareEntry describes one group share area from the current user's perspective.
	GroupShareEntry struct {
		ID           string `json:"id"`                 // group hashid
		Name         string `json:"name"`               // group name
		Status       string `json:"status"`             // owner | member | pending | joinable
		IsApprover   bool   `json:"is_approver"`        // current user can approve join requests
		PendingCount int    `json:"pending_count"`      // pending applications (approver only)
		Uri          string `json:"uri,omitempty"`      // cloudreve://<gid>@group, set when accessible
	}

	GroupShareListResponse struct {
		Groups []GroupShareEntry `json:"groups"`
	}

	// GroupShareIDService addresses a single group share by its hashid.
	GroupShareIDService struct {
		ID string `uri:"id" binding:"required"`
	}
	GroupShareIDParamCtx struct{}

	// ApplyGroupShareService submits a join application with the applicant's real name and reason.
	// ID is taken from the path param by the controller (FromJSON does not bind URI).
	ApplyGroupShareService struct {
		ID       string `json:"-"`
		RealName string `json:"real_name" binding:"required,max=255"`
		Reason   string `json:"reason" binding:"required,max=1000"`
	}
	ApplyGroupShareParamCtx struct{}

	// GroupShareApplicant is a pending applicant with the info the approver needs.
	GroupShareApplicant struct {
		User     User   `json:"user"`
		RealName string `json:"real_name"`
		Reason   string `json:"reason"`
	}

	// ReviewGroupShareService approves or rejects a pending applicant.
	// ID is taken from the path param by the controller (FromJSON does not bind URI).
	ReviewGroupShareService struct {
		ID      string `json:"-"`
		UserID  string `json:"user_id" binding:"required"`
		Approve bool   `json:"approve"`
	}
	ReviewGroupShareParamCtx struct{}
)

func groupShareUri(hasher hashid.Encoder, groupID int) string {
	return fmt.Sprintf("%s://%s@%s", constants.CloudreveScheme, hashid.EncodeGroupID(hasher, groupID), constants.FileSystemGroup)
}

// ListGroupShares lists every group share area with the current user's relationship to it.
func ListGroupShares(c *gin.Context) (*GroupShareListResponse, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	if u == nil || inventory.IsAnonymousUser(u) {
		return nil, serializer.NewError(serializer.CodeCheckLogin, "Login required", nil)
	}
	hasher := dep.HashIDEncoder()

	groups, err := dep.GroupClient().ListAll(c)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list groups", err)
	}

	res := &GroupShareListResponse{Groups: make([]GroupShareEntry, 0)}
	for _, g := range groups {
		if !inventory.IsShareGroup(g) {
			continue
		}

		entry := GroupShareEntry{
			ID:   hashid.EncodeGroupID(hasher, g.ID),
			Name: g.Name,
		}

		isOwner := g.Settings.ShareRootOwner == u.ID
		isAdmin := u.Edges.Group != nil && u.Edges.Group.Permissions != nil &&
			u.Edges.Group.Permissions.Enabled(int(types.GroupPermissionIsAdmin))

		switch {
		case isOwner:
			entry.Status = GroupShareStatusOwner
		case inventory.CanAccessGroupShare(g, u):
			entry.Status = GroupShareStatusMember
		case hasPendingRequest(g.Settings.ShareJoinRequests, u.ID):
			entry.Status = GroupShareStatusPending
		default:
			entry.Status = GroupShareStatusJoinable
		}

		if isOwner || isAdmin {
			entry.IsApprover = true
			entry.PendingCount = len(g.Settings.ShareJoinRequests)
		}

		if entry.Status == GroupShareStatusOwner || entry.Status == GroupShareStatusMember {
			entry.Uri = groupShareUri(hasher, g.ID)
		}

		res.Groups = append(res.Groups, entry)
	}

	return res, nil
}

// loadShareGroup decodes the group hashid and returns the group, ensuring it offers a share area.
func loadShareGroup(c context.Context, dep dependency.Dep, idHash string) (*ent.Group, error) {
	gid, err := dep.HashIDEncoder().Decode(idHash, hashid.GroupID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid group id", err)
	}
	g, err := dep.GroupClient().GetByID(c, gid)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Group not found", err)
	}
	if !inventory.IsShareGroup(g) {
		return nil, serializer.NewError(serializer.CodeNotFound, "Group share area not available", nil)
	}
	return g, nil
}

// hasPendingRequest reports whether the given user has a pending application.
func hasPendingRequest(reqs []types.GroupJoinRequest, uid int) bool {
	return lo.ContainsBy(reqs, func(r types.GroupJoinRequest) bool { return r.UserID == uid })
}

// Apply submits a request to join the group share area with a real name and reason.
func (s *ApplyGroupShareService) Apply(c *gin.Context) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	if u == nil || inventory.IsAnonymousUser(u) {
		return serializer.NewError(serializer.CodeCheckLogin, "Login required", nil)
	}

	g, err := loadShareGroup(c, dep, s.ID)
	if err != nil {
		return err
	}

	if inventory.CanAccessGroupShare(g, u) {
		return serializer.NewError(serializer.CodeParamErr, "You can already access this group share", nil)
	}

	req := types.GroupJoinRequest{UserID: u.ID, RealName: s.RealName, Reason: s.Reason}
	if hasPendingRequest(g.Settings.ShareJoinRequests, u.ID) {
		// Update the existing pending application instead of duplicating it.
		g.Settings.ShareJoinRequests = lo.Map(g.Settings.ShareJoinRequests, func(r types.GroupJoinRequest, _ int) types.GroupJoinRequest {
			if r.UserID == u.ID {
				return req
			}
			return r
		})
	} else {
		g.Settings.ShareJoinRequests = append(g.Settings.ShareJoinRequests, req)
	}

	if err := dep.GroupClient().SaveSettings(c, g); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to submit application", err)
	}
	return nil
}

// requireApprover ensures the current user may approve requests for the group (owner or admin).
func requireApprover(c *gin.Context, g *ent.Group) error {
	u := inventory.UserFromContext(c)
	if u == nil {
		return serializer.NewError(serializer.CodeCheckLogin, "Login required", nil)
	}
	isOwner := g.Settings.ShareRootOwner == u.ID
	isAdmin := u.Edges.Group != nil && u.Edges.Group.Permissions != nil &&
		u.Edges.Group.Permissions.Enabled(int(types.GroupPermissionIsAdmin))
	if !isOwner && !isAdmin {
		return serializer.NewError(serializer.CodeNoPermissionErr, "Only the share area owner can do this", nil)
	}
	return nil
}

// ListApplications returns the pending applicants of the group share area (approver only),
// including each applicant's submitted real name and reason.
func (s *GroupShareIDService) ListApplications(c *gin.Context) ([]GroupShareApplicant, error) {
	dep := dependency.FromContext(c)
	g, err := loadShareGroup(c, dep, s.ID)
	if err != nil {
		return nil, err
	}
	if err := requireApprover(c, g); err != nil {
		return nil, err
	}

	hasher := dep.HashIDEncoder()
	res := make([]GroupShareApplicant, 0, len(g.Settings.ShareJoinRequests))
	for _, req := range g.Settings.ShareJoinRequests {
		applicant, err := dep.UserClient().GetByID(c, req.UserID)
		if err != nil {
			continue
		}
		res = append(res, GroupShareApplicant{
			User:     BuildUserRedacted(c, applicant, RedactLevelUser, hasher),
			RealName: req.RealName,
			Reason:   req.Reason,
		})
	}
	return res, nil
}

// Review approves or rejects a pending applicant (approver only).
func (s *ReviewGroupShareService) Review(c *gin.Context) error {
	dep := dependency.FromContext(c)
	g, err := loadShareGroup(c, dep, s.ID)
	if err != nil {
		return err
	}
	if err := requireApprover(c, g); err != nil {
		return err
	}

	applicantID, err := dep.HashIDEncoder().Decode(s.UserID, hashid.UserID)
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "Invalid user id", err)
	}

	if !hasPendingRequest(g.Settings.ShareJoinRequests, applicantID) {
		return serializer.NewError(serializer.CodeNotFound, "Application not found", nil)
	}

	// Remove from pending; add to members when approved.
	g.Settings.ShareJoinRequests = lo.Filter(g.Settings.ShareJoinRequests, func(r types.GroupJoinRequest, _ int) bool {
		return r.UserID != applicantID
	})
	if s.Approve && !lo.Contains(g.Settings.ShareMembers, applicantID) {
		g.Settings.ShareMembers = append(g.Settings.ShareMembers, applicantID)
	}

	if err := dep.GroupClient().SaveSettings(c, g); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to update application", err)
	}
	return nil
}
