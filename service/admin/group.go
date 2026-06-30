package admin

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

// groupShareRootName returns the (orphan folder) name used for a group's shared file area root.
func groupShareRootName(groupID int) string {
	return fmt.Sprintf("__group_share_%d__", groupID)
}

// AddGroupService 用户组添加服务
type AddGroupService struct {
	//Group model.Group `json:"group" binding:"required"`
}

// GroupService 用户组ID服务
type GroupService struct {
	ID uint `uri:"id" json:"id" binding:"required"`
}

// Get 获取用户组详情
func (service *GroupService) Get() serializer.Response {
	//group, err := model.GetGroupByID(service.ID)
	//if err != nil {
	//	return serializer.ErrDeprecated(serializer.CodeGroupNotFound, "", err)
	//}
	//
	//return serializer.Response{Data: group}

	return serializer.Response{}
}

// Delete 删除用户组
func (service *GroupService) Delete() serializer.Response {
	//// 查找用户组
	//group, err := model.GetGroupByID(service.ID)
	//if err != nil {
	//	return serializer.ErrDeprecated(serializer.CodeGroupNotFound, "", err)
	//}
	//
	//// 是否为系统用户组
	//if group.ID <= 3 {
	//	return serializer.ErrDeprecated(serializer.CodeInvalidActionOnSystemGroup, "", err)
	//}
	//
	//// 检查是否有用户使用
	//total := 0
	//row := model.DB.Model(&model.User{}).Where("group_id = ?", service.ID).
	//	Select("count(id)").Row()
	//row.Scan(&total)
	//if total > 0 {
	//	return serializer.ErrDeprecated(serializer.CodeGroupUsedByUser, strconv.Itoa(total), nil)
	//}
	//
	//model.DB.Delete(&group)

	return serializer.Response{}
}

func (service *SingleGroupService) Delete(c *gin.Context) error {
	if service.ID <= 3 {
		return serializer.NewError(serializer.CodeInvalidActionOnSystemGroup, "", nil)
	}

	dep := dependency.FromContext(c)
	groupClient := dep.GroupClient()

	// Any user still under this group?
	users, err := groupClient.CountUsers(c, int(service.ID))
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to count users", err)
	}

	if users > 0 {
		return serializer.NewError(serializer.CodeGroupUsedByUser, strconv.Itoa(users), nil)
	}

	err = groupClient.Delete(c, service.ID)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete group", err)
	}

	return nil
}

func (s *AdminListService) List(c *gin.Context) (*ListGroupResponse, error) {
	dep := dependency.FromContext(c)
	groupClient := dep.GroupClient()

	ctx := context.WithValue(c, inventory.LoadGroupPolicy{}, true)
	res, err := groupClient.ListGroups(ctx, &inventory.ListGroupParameters{
		PaginationArgs: &inventory.PaginationArgs{
			Page:     s.Page - 1,
			PageSize: s.PageSize,
			OrderBy:  s.OrderBy,
			Order:    inventory.OrderDirection(s.OrderDirection),
		},
	})

	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list groups", err)
	}

	return &ListGroupResponse{
		Pagination: res.PaginationResults,
		Groups:     res.Groups,
	}, nil
}

type (
	SingleGroupService struct {
		ID int `uri:"id" json:"id" binding:"required"`
	}
	SingleGroupParamCtx struct{}
)

const (
	countUserQuery = "countUser"
)

func (s *SingleGroupService) Get(c *gin.Context) (*GetGroupResponse, error) {
	dep := dependency.FromContext(c)
	groupClient := dep.GroupClient()

	ctx := context.WithValue(c, inventory.LoadGroupPolicy{}, true)
	group, err := groupClient.GetByID(ctx, s.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get group", err)
	}

	res := &GetGroupResponse{Group: group}

	if c.Query(countUserQuery) != "" {
		totalUsers, err := groupClient.CountUsers(ctx, int(s.ID))
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to count users", err)
		}
		res.TotalUsers = totalUsers
	}

	return res, nil
}

type (
	UpsertGroupService struct {
		Group *ent.Group `json:"group" binding:"required"`
		// ShareRootOwnerHash is the hashid of the user to set as the group share root owner.
		// nil = leave unchanged; "" = clear; otherwise decoded into Group.Settings.ShareRootOwner.
		ShareRootOwnerHash *string `json:"share_root_owner_hash,omitempty"`
	}
	UpsertGroupParamCtx struct{}
)

func (s *UpsertGroupService) Update(c *gin.Context) (*GetGroupResponse, error) {
	dep := dependency.FromContext(c)
	groupClient := dep.GroupClient()

	if s.Group.ID == 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "ID is required", nil)
	}

	// Initial admin group have to be admin
	if s.Group.ID == 1 && !s.Group.Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
		return nil, serializer.NewError(serializer.CodeParamErr, "Initial admin group have to be admin", nil)
	}

	if s.Group.Settings == nil {
		s.Group.Settings = &types.GroupSetting{}
	}

	// Resolve the desired share root owner from its hashid (the frontend only knows hashids).
	if s.ShareRootOwnerHash != nil {
		if *s.ShareRootOwnerHash == "" {
			s.Group.Settings.ShareRootOwner = 0
		} else {
			ownerID, err := dep.HashIDEncoder().Decode(*s.ShareRootOwnerHash, hashid.UserID)
			if err != nil {
				return nil, serializer.NewError(serializer.CodeParamErr, "Invalid share root owner", err)
			}
			s.Group.Settings.ShareRootOwner = ownerID
		}
	}

	// Load existing group to learn the current share-root state, which is managed internally
	// and must not be clobbered by the incoming payload.
	existing, err := groupClient.GetByID(c, s.Group.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get group", err)
	}
	var oldRootID, oldRootOwner int
	if existing.Settings != nil {
		oldRootID, oldRootOwner = existing.Settings.ShareRootID, existing.Settings.ShareRootOwner
	}

	desiredOwner := s.Group.Settings.ShareRootOwner
	// The share root ID is internal; always carry the existing one forward.
	s.Group.Settings.ShareRootID = oldRootID

	// Validate the desired owner up-front (read-only checks before opening a transaction).
	if desiredOwner > 0 {
		owner, err := dep.UserClient().GetByID(c, desiredOwner)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Share root owner not found", err)
		}
		if owner.Status != user.StatusActive {
			return nil, serializer.NewError(serializer.CodeParamErr, "Share root owner is not active", nil)
		}
	}

	// Reconcile the group share root (create / migrate owner) and persist the group atomically.
	fc, tx, ctx, err := inventory.WithTx(c, dep.FileClient())
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	if desiredOwner > 0 {
		if s.Group.Settings.ShareRootID == 0 {
			// First time: create the shared root folder owned by the configured owner.
			root, err := fc.CreateFolder(ctx, nil, &inventory.CreateFolderParameters{
				Owner: desiredOwner,
				Name:  groupShareRootName(s.Group.ID),
			})
			if err != nil {
				_ = inventory.Rollback(tx)
				return nil, serializer.NewError(serializer.CodeDBError, "Failed to create group share root", err)
			}

			// Mark the orphan root so it stays hidden from the owner's trash listing.
			if _, err := fc.UpdateProps(ctx, root, &types.FileProps{GroupShareRoot: true}); err != nil {
				_ = inventory.Rollback(tx)
				return nil, serializer.NewError(serializer.CodeDBError, "Failed to mark group share root", err)
			}

			s.Group.Settings.ShareRootID = root.ID
		} else if oldRootOwner != desiredOwner {
			// Owner changed: migrate the whole subtree's ownership and move its storage usage.
			root, err := fc.GetByID(ctx, s.Group.Settings.ShareRootID)
			if err != nil {
				_ = inventory.Rollback(tx)
				return nil, serializer.NewError(serializer.CodeDBError, "Failed to load group share root", err)
			}

			from := oldRootOwner
			if from == 0 {
				from = root.OwnerID
			}
			diff, err := fc.TransferDescendantsOwner(ctx, root, from, desiredOwner)
			if err != nil {
				_ = inventory.Rollback(tx)
				return nil, serializer.NewError(serializer.CodeDBError, "Failed to transfer group share ownership", err)
			}
			tx.AppendStorageDiff(diff)
		}
	}
	s.Group.Settings.ShareRootOwner = desiredOwner

	txGroupClient, _ := inventory.InheritTx(ctx, groupClient)
	group, err := txGroupClient.Upsert(ctx, s.Group)
	if err != nil {
		_ = inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update group", err)
	}

	if err := inventory.CommitWithStorageDiff(ctx, tx, dep.Logger(), dep.UserClient()); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to commit group update", err)
	}

	service := &SingleGroupService{ID: group.ID}
	return service.Get(c)
}

func (s *UpsertGroupService) Create(c *gin.Context) (*GetGroupResponse, error) {
	dep := dependency.FromContext(c)
	groupClient := dep.GroupClient()

	if s.Group.ID > 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "ID must be 0", nil)
	}

	group, err := groupClient.Upsert(c, s.Group)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create group", err)
	}

	service := &SingleGroupService{ID: group.ID}
	return service.Get(c)
}
