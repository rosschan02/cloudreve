package dbfs

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

var groupNavigatorCapability = &boolset.BooleanSet{}

// ErrGroupShareNotConfigured is returned when the group share area has no owner / root configured.
var ErrGroupShareNotConfigured = serializer.NewError(serializer.CodeNotFound, "Group share area is not configured yet", nil)

// NewGroupNavigator creates a navigator for the user group's shared file area.
// All members of a group share a single root folder owned by the group's configured
// "share root owner" (an administrator). Storage of uploaded files is charged to that owner.
func NewGroupNavigator(u *ent.User, fileClient inventory.FileClient, userClient inventory.UserClient, l logging.Logger,
	config *setting.DBFS, hasher hashid.Encoder) Navigator {
	return &groupNavigator{
		user:          u,
		l:             l,
		fileClient:    fileClient,
		userClient:    userClient,
		config:        config,
		baseNavigator: newBaseNavigator(fileClient, defaultFilter, u, hasher, config),
	}
}

type groupNavigator struct {
	l          logging.Logger
	user       *ent.User
	fileClient inventory.FileClient
	userClient inventory.UserClient

	config *setting.DBFS
	*baseNavigator
	root           *File
	disableRecycle bool
	persist        func()
}

func (n *groupNavigator) Recycle() {
	if n.persist != nil {
		n.persist()
		n.persist = nil
	}
	if n.root != nil && !n.disableRecycle {
		n.root.Recycle()
	}
}

func (n *groupNavigator) PersistState(kv cache.Driver, key string) {
	n.disableRecycle = true
	n.persist = func() {
		kv.Set(key, n.root, ContextHintTTL)
	}
}

func (n *groupNavigator) RestoreState(s State) error {
	n.disableRecycle = true
	if state, ok := s.(*File); ok {
		n.root = state
		return nil
	}

	return fmt.Errorf("invalid state type: %T", s)
}

func (n *groupNavigator) To(ctx context.Context, path *fs.URI) (*File, error) {
	if n.root == nil {
		// Anonymous user does not have a group share area.
		if inventory.IsAnonymousUser(n.user) {
			return nil, ErrLoginRequired
		}

		group := n.user.Edges.Group
		if group == nil {
			return nil, ErrPermissionDenied.WithError(fmt.Errorf("user group not loaded"))
		}

		// Access to the group share area is gated by a dedicated group permission.
		if !group.Permissions.Enabled(int(types.GroupPermissionAccessGroupShare)) {
			return nil, ErrPermissionDenied.WithError(fmt.Errorf("group share access not granted"))
		}

		settings := group.Settings
		if settings == nil || settings.ShareRootOwner == 0 || settings.ShareRootID == 0 {
			return nil, ErrGroupShareNotConfigured
		}

		// Load the configured owner of the share root (storage quota is charged here).
		ctx = context.WithValue(ctx, inventory.LoadUserGroup{}, true)
		owner, err := n.userClient.GetByID(ctx, settings.ShareRootOwner)
		if err != nil {
			return nil, ErrGroupShareNotConfigured.WithError(fmt.Errorf("share root owner not found: %w", err))
		}
		if owner.Status != user.StatusActive {
			return nil, ErrGroupShareNotConfigured.WithError(fmt.Errorf("share root owner is not active"))
		}

		rootFile, err := n.fileClient.GetByID(ctx, settings.ShareRootID)
		if err != nil {
			n.l.Info("Group share root folder not found: %s", err)
			// Do not return ErrFsNotInitialized here: that would make the caller try to
			// initialize the *current user's* personal root as a fallback.
			return nil, ErrGroupShareNotConfigured.WithError(fmt.Errorf("share root folder missing: %w", err))
		}

		// Sanity check: the root folder must belong to the configured owner.
		if rootFile.OwnerID != owner.ID {
			return nil, ErrGroupShareNotConfigured.WithError(fmt.Errorf("share root owner mismatch"))
		}

		n.root = newFile(nil, rootFile)
		rootPath := newGroupUri("")
		n.root.Path[pathIndexRoot], n.root.Path[pathIndexUser] = rootPath, rootPath
		n.root.OwnerModel = owner
		n.root.IsUserRoot = true
		n.root.CapabilitiesBs = n.Capabilities(false).Capability
	}

	current, lastAncestor := n.root, n.root
	elements := path.Elements()
	var err error
	for index, element := range elements {
		lastAncestor = current
		current, err = n.walkNext(ctx, current, element, index == len(elements)-1)
		if err != nil {
			return lastAncestor, fmt.Errorf("failed to walk into %q: %w", element, err)
		}
	}

	return current, nil
}

func (n *groupNavigator) Children(ctx context.Context, parent *File, args *ListArgs) (*ListResult, error) {
	return n.baseNavigator.children(ctx, parent, args)
}

func (n *groupNavigator) walkNext(ctx context.Context, root *File, next string, isLeaf bool) (*File, error) {
	return n.baseNavigator.walkNext(ctx, root, next, isLeaf)
}

func (n *groupNavigator) Capabilities(isSearching bool) *fs.NavigatorProps {
	res := &fs.NavigatorProps{
		Capability:            groupNavigatorCapability,
		OrderDirectionOptions: fullOrderDirectionOption,
		OrderByOptions:        fullOrderByOption,
		MaxPageSize:           n.config.MaxPageSize,
	}
	if isSearching {
		res.OrderByOptions = nil
		res.OrderDirectionOptions = nil
	}

	return res
}

func (n *groupNavigator) Walk(ctx context.Context, levelFiles []*File, limit, depth int, f WalkFunc) error {
	return n.baseNavigator.walk(ctx, levelFiles, limit, depth, f)
}

func (n *groupNavigator) FollowTx(ctx context.Context) (func(), error) {
	if _, ok := ctx.Value(inventory.TxCtx{}).(*inventory.Tx); !ok {
		return nil, fmt.Errorf("navigator: no inherited transaction found in context")
	}
	newFileClient, _, _, err := inventory.WithTx(ctx, n.fileClient)
	if err != nil {
		return nil, err
	}

	newUserClient, _, _, err := inventory.WithTx(ctx, n.userClient)
	if err != nil {
		return nil, err
	}

	oldFileClient, oldUserClient := n.fileClient, n.userClient
	revert := func() {
		n.fileClient = oldFileClient
		n.userClient = oldUserClient
		n.baseNavigator.fileClient = oldFileClient
	}

	n.fileClient = newFileClient
	n.userClient = newUserClient
	n.baseNavigator.fileClient = newFileClient
	return revert, nil
}

func (n *groupNavigator) ExecuteHook(ctx context.Context, hookType fs.HookType, file *File) error {
	return nil
}

func (n *groupNavigator) GetView(ctx context.Context, file *File) *types.ExplorerView {
	return file.View()
}
