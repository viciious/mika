package memory

import (
	"github.com/viciious/mika/config"
	"github.com/viciious/mika/consts"
	"github.com/viciious/mika/store"
	"github.com/pkg/errors"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	driverName = "memory"
)

func (d *Driver) Name() string {
	return driverName
}

func (d *Driver) TorrentSave(_ *store.Torrent) error {
	return nil
}

// Add adds a new torrent to the memory store
func (d *Driver) TorrentAdd(t *store.Torrent) error {
	d.torrentsMu.Lock()
	defer d.torrentsMu.Unlock()
	_, found := d.torrents[t.InfoHash]
	if found {
		return consts.ErrDuplicate
	}
	d.torrents[t.InfoHash] = t
	return nil
}

// Delete will mark a torrent as deleted in the backing store.
// NOTE the memory store always permanently deletes the torrent
func (d *Driver) TorrentDelete(ih store.InfoHash, _ bool) error {
	d.torrentsMu.Lock()
	delete(d.torrents, ih)
	d.torrentsMu.Unlock()
	return nil
}

// Sync batch updates the backing store with the new TorrentStats provided
func (d *Driver) TorrentSync(_ []*store.Torrent) error {
	return nil
}

// Conn always returns nil for in-memory store
func (d *Driver) Conn() interface{} {
	return nil
}

// WhiteListDelete removes a client from the global whitelist
func (d *Driver) WhiteListDelete(client *store.WhiteListClient) error {
	d.whitelistMu.Lock()
	defer d.whitelistMu.Unlock()
	delete(d.whitelist, client.ClientPrefix)
	return nil
}

// WhiteListAdd will insert a new client prefix into the allowed clients list
func (d *Driver) WhiteListAdd(client *store.WhiteListClient) error {
	d.whitelistMu.Lock()
	defer d.whitelistMu.Unlock()
	d.whitelist[client.ClientPrefix] = client
	return nil
}

// WhiteListGetAll fetches all known whitelisted clients
func (d *Driver) WhiteListGetAll() ([]*store.WhiteListClient, error) {
	d.whitelistMu.RLock()
	defer d.whitelistMu.RUnlock()
	var wl []*store.WhiteListClient
	for _, wlc := range d.whitelist {
		wl = append(wl, wlc)
	}
	return wl, nil
}

// Get returns the Torrent matching the infohash
func (d *Driver) TorrentGet(hash store.InfoHash, deletedOk bool) (*store.Torrent, error) {
	d.torrentsMu.RLock()
	t, found := d.torrents[hash]
	d.torrentsMu.RUnlock()
	if !found {
		return nil, consts.ErrInvalidInfoHash
	}
	if t.IsDeleted && !deletedOk {
		return nil, consts.ErrInvalidInfoHash
	}
	return t, nil
}

// NewPeerStore instantiates a new in-memory peer store
func NewDriver() *Driver {
	return &Driver{
		users:       make(store.Users),
		roles:       make(store.Roles),
		torrents:    make(store.Torrents),
		whitelist:   make(store.WhiteList),
		rolesMu:     &sync.RWMutex{},
		torrentsMu:  &sync.RWMutex{},
		usersMu:     &sync.RWMutex{},
		whitelistMu: &sync.RWMutex{},
	}
}

// Driver is the memory backed store.Store implementation
type Driver struct {
	users       store.Users
	roles       store.Roles
	torrents    store.Torrents
	whitelist   store.WhiteList
	rolesMu     *sync.RWMutex
	torrentsMu  *sync.RWMutex
	usersMu     *sync.RWMutex
	whitelistMu *sync.RWMutex
	lastUserID  uint32
	lastRoleID  uint32
}

func (d *Driver) Migrate() error {
	return nil
}

func (d *Driver) Users() (store.Users, error) {
	return d.users, nil
}

func (d *Driver) Torrents() (store.Torrents, error) {
	return d.torrents, nil
}

func (d *Driver) RoleSave(r *store.Role) error {
	if r.RoleID == 0 {
		return d.RoleAdd(r)
	}
	return nil
}

func (d *Driver) RoleByID(roleID uint32) (*store.Role, error) {
	for _, r := range d.roles {
		if r.RoleID == roleID {
			return r, nil
		}
	}
	return nil, consts.ErrInvalidRole
}

func (d *Driver) RoleAdd(role *store.Role) error {
	d.rolesMu.Lock()
	defer d.rolesMu.Unlock()
	atomic.AddUint32(&role.RoleID, atomic.AddUint32(&d.lastRoleID, 1))
	for _, r := range d.roles {
		if strings.ToLower(r.RoleName) == strings.ToLower(role.RoleName) {
			return errors.Errorf("duplicate role_name: %s", role.RoleName)
		}
		if r.RoleID == role.RoleID {
			return errors.Errorf("duplicate role_Id: %d", r.RoleID)
		}
	}
	d.roles[role.RoleID] = role
	return nil
}

func (d *Driver) RoleDelete(roleID uint32) error {
	d.rolesMu.Lock()
	defer d.rolesMu.Unlock()
	conflicts := 0
	for _, u := range d.users {
		if u.RoleID == roleID {
			conflicts++
		}
	}
	if conflicts > 0 {
		return errors.Errorf("Found %d users with only a single role, cannot remove only role", conflicts)
	}
	delete(d.roles, roleID)
	return nil
}

func (d *Driver) Roles() (store.Roles, error) {
	return d.roles, nil
}

// Update is used to change a known user
func (d *Driver) UserSave(u *store.User) error {
	return nil
}

// Sync batch updates the backing store with the new UserStats provided
func (d *Driver) UserSync(_ []*store.User) error {
	return nil
}

// Add will add a new user to the backing store
func (d *Driver) UserAdd(usr *store.User) error {
	d.usersMu.Lock()
	defer d.usersMu.Unlock()
	atomic.AddUint32(&usr.UserID, atomic.AddUint32(&d.lastUserID, 1))
	for _, existing := range d.users {
		if existing.UserID == usr.UserID {
			return consts.ErrDuplicate
		}
	}
	d.users[usr.Passkey] = usr
	return nil
}

// GetByPasskey will lookup and return the user via their passkey used as an identifier
// The errors returned for this method should be very generic and not reveal any info
// that could possibly help attackers gain any insight. All error cases MUST
// return ErrUnauthorized.
func (d *Driver) UserGetByPasskey(passkey string) (*store.User, error) {
	d.usersMu.RLock()
	user, found := d.users[passkey]
	d.usersMu.RUnlock()
	if !found {
		return nil, consts.ErrUnauthorized
	}
	return user, nil
}

// GetByID returns a user matching the userId
func (d *Driver) UserGetByID(userID uint32) (*store.User, error) {
	d.usersMu.RLock()
	defer d.usersMu.RUnlock()
	for _, usr := range d.users {
		if usr.UserID == userID {
			return usr, nil
		}
	}
	return nil, consts.ErrUnauthorized
}

// Delete removes a user from the backing store
func (d *Driver) UserDelete(user *store.User) error {
	d.usersMu.Lock()
	delete(d.users, user.Passkey)
	d.usersMu.Unlock()
	return nil
}

// Close will delete/free the underlying memory store
func (d *Driver) Close() error {
	d.usersMu.Lock()
	d.users = make(map[string]*store.User)
	d.usersMu.Unlock()
	d.torrentsMu.Lock()
	d.torrents = make(store.Torrents)
	d.torrentsMu.Unlock()
	return nil
}

type initializer struct{}

// New creates a new memory backed user store.
func (d initializer) New(_ config.StoreConfig) (store.Store, error) {
	return NewDriver(), nil
}

func init() {
	store.AddDriver(driverName, initializer{})
}
