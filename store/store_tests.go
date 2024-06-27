package store

import (
	"github.com/leighmacdonald/golib"
	"github.com/viciious/mika/consts"
	"github.com/viciious/mika/util"
	"github.com/stretchr/testify/require"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"
)

// GenerateTestRole creates a role using fake data. Used for testing.
func GenerateTestRole() Role {
	return Role{
		RoleName:        golib.RandomString(10),
		Priority:        100,
		MultiUp:         1.0,
		MultiDown:       1.0,
		DownloadEnabled: true,
		UploadEnabled:   true,
		CreatedOn:       util.Now(),
		UpdatedOn:       util.Now(),
	}
}

// GenerateTestUser creates a peer using fake data. Used for testing.
func GenerateTestUser() User {
	return User{
		Passkey:         util.NewPasskey(),
		IsDeleted:       false,
		DownloadEnabled: true,
		Downloaded:      1000,
		Uploaded:        2000,
		Announces:       500,
	}
}

// GenerateTestTorrent creates a torrent using fake data. Used for testing.
func GenerateTestTorrent() Torrent {
	token, _ := util.GenRandomBytes(20)
	var ih InfoHash
	if err := InfoHashFromBytes(&ih, token); err != nil {
		log.Panicf("Failed to generate info_hash: %s", err.Error())
	}
	return NewTorrent(ih)
}

// GenerateTestPeer creates a peer using fake data for the provided user. Used for testing.
func GenerateTestPeer() *Peer {
	token, _ := util.GenRandomBytes(20)
	pid := PeerIDFromString(string(token))
	p := NewPeer(
		uint32(rand.Intn(1000000)),
		pid,
		net.ParseIP("1.2.3.4"),
		uint16(rand.Intn(60000)))
	return p
}

// TestTorrentStore tests the interface implementation
func TestStore(t *testing.T, s Store) {
	torrentA := GenerateTestTorrent()
	require.NoError(t, s.TorrentAdd(&torrentA))
	//fetchedTorrent, err := s.TorrentGet(torrentA.InfoHash, false)
	//require.NoError(t, err)
	//require.Equal(t, torrentA.InfoHash, fetchedTorrent.InfoHash)
	//require.Equal(t, torrentA.IsDeleted, fetchedTorrent.IsDeleted)
	//require.Equal(t, torrentA.IsEnabled, fetchedTorrent.IsEnabled)
	//batch := []*Torrent{{
	//	Seeders:    uint32(rand.Intn(100000)),
	//	Leechers:   uint32(rand.Intn(100000)),
	//	Snatches:   uint32(rand.Intn(10000)),
	//	Uploaded:   uint64(rand.Intn(100000)),
	//	Downloaded: uint64(rand.Intn(100000)),
	//	Announces:  uint64(rand.Intn(100000)),
	//},
	//}
	//require.NoError(t, s.TorrentSync(batch), "[%s] Failed to sync torrent", s.Name())
	//updated, err := s.TorrentGet(torrentA.InfoHash, false)
	//require.NoError(t, err)
	//require.Equal(t, torrentA.Seeders+batch[0].Seeders, updated.Seeders)
	//require.Equal(t, torrentA.Leechers+batch[0].Leechers, updated.Leechers)
	//require.Equal(t, torrentA.Snatches+batch[0].Snatches, updated.Snatches)
	//require.Equal(t, torrentA.Uploaded+batch[0].Uploaded, updated.Uploaded)
	//require.Equal(t, torrentA.Downloaded+batch[0].Downloaded, updated.Downloaded)
	//require.Equal(t, torrentA.Announces+batch[0].Announces, updated.Announces)

	require.NoError(t, s.TorrentDelete(torrentA.InfoHash, true))
	_, err := s.TorrentGet(torrentA.InfoHash, false)
	require.Equal(t, consts.ErrInvalidInfoHash, err)
	wlClients := []*WhiteListClient{
		{ClientPrefix: "UT", ClientName: "uTorrent"},
		{ClientPrefix: "qT", ClientName: "QBittorrent"},
	}
	for _, c := range wlClients {
		require.NoError(t, s.WhiteListAdd(c))
	}
	clients, err3 := s.WhiteListGetAll()
	require.NoError(t, err3)
	require.Equal(t, len(wlClients), len(clients))
	require.NoError(t, s.WhiteListDelete(wlClients[0]))
	clientsUpdated, _ := s.WhiteListGetAll()
	require.Equal(t, len(wlClients)-1, len(clientsUpdated))

	roles := []*Role{
		{
			RoleName:        "Admin",
			Priority:        1,
			MultiUp:         1.0,
			MultiDown:       1.0,
			DownloadEnabled: true,
			UploadEnabled:   true,
		},
		{
			RoleName:        "Moderator",
			Priority:        5,
			MultiUp:         1.0,
			MultiDown:       1.0,
			DownloadEnabled: true,
			UploadEnabled:   true,
		},
		{
			RoleName:        "Member",
			Priority:        15,
			MultiUp:         1.0,
			MultiDown:       1.0,
			DownloadEnabled: true,
			UploadEnabled:   true,
		},
		{
			RoleName:        "Master",
			Priority:        10,
			MultiUp:         1.1,
			MultiDown:       1.0,
			DownloadEnabled: true,
			UploadEnabled:   true,
		},
	}
	for _, role := range roles {
		require.NoError(t, s.RoleAdd(role), "Failed to add role")
	}
	fetchedRoles, err := s.Roles()
	require.NoError(t, err, "failed to fetch roles")
	require.Equal(t, len(roles), len(fetchedRoles))
	require.NoError(t, s.RoleDelete(roles[3].RoleID))
	fetchedRolesDeleted, err := s.Roles()
	require.NoError(t, err, "failed to fetch roles")
	require.Equal(t, len(roles)-1, len(fetchedRolesDeleted))

	var users []*User
	for i := 0; i < 3; i++ {
		usr := GenerateTestUser()
		r := roles[uint32(i)]
		usr.RoleID = r.RoleID
		users = append(users, &usr)
	}
	if users == nil {
		t.Fatalf("[%s] Failed to setup users", s.Name())
	}
	require.NoError(t, s.UserAdd(users[0]))
	fetchedUserID, err := s.UserGetByID(users[0].UserID)
	require.NoError(t, err)
	require.Equal(t, users[0].RoleID, fetchedUserID.RoleID)

	fetchedUserPasskey, err := s.UserGetByPasskey(users[0].Passkey)
	require.NoError(t, err)
	require.Equal(t, users[0].RoleID, fetchedUserPasskey.RoleID)

	//batchUpdate := []*User{
	//	{
	//		Passkey:    users[0].Passkey,
	//		Uploaded:   1000,
	//		Downloaded: 2000,
	//		Announces:  10,
	//	},
	//}
	//require.NoError(t, s.UserSync(batchUpdate))
	//time.Sleep(100 * time.Millisecond)
	//var updatedUser User
	//require.NoError(t, s.UserGetByPasskey(&updatedUser, users[0].Passkey))
	//require.Equal(t, uint64(1000)+users[0].Uploaded, updatedUser.Uploaded)
	//require.Equal(t, uint64(2000)+users[0].Downloaded, updatedUser.Downloaded)
	//require.Equal(t, uint32(10)+users[0].Announces, updatedUser.Announces)

	newUser := GenerateTestUser()
	newUser.RoleID = roles[0].RoleID
	require.NoError(t, s.UserAdd(&newUser))
	fetchedNewUser, err := s.UserGetByID(newUser.UserID)
	require.NoError(t, err)
	require.Equal(t, newUser.UserID, fetchedNewUser.UserID)
	require.Equal(t, newUser.Passkey, fetchedNewUser.Passkey)
	require.Equal(t, newUser.IsDeleted, fetchedNewUser.IsDeleted)
	require.Equal(t, newUser.DownloadEnabled, fetchedNewUser.DownloadEnabled)
	require.Equal(t, newUser.Downloaded, fetchedNewUser.Downloaded)
	require.Equal(t, newUser.Uploaded, fetchedNewUser.Uploaded)
	require.Equal(t, newUser.Announces, fetchedNewUser.Announces)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
