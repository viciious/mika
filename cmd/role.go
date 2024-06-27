package cmd

import (
	"context"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/viciious/mika/client"
	pb "github.com/viciious/mika/proto"
	"github.com/viciious/mika/rpc"
	"github.com/viciious/mika/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"os"
)

var (
	roleAddParam  = &pb.RoleAddParams{}
	roleDelParam  = &pb.RoleID{}
	roleSetParams = &pb.RoleSetParams{
		UpdatedKeys:     nil,
		RoleName:        "",
		RemoteId:        0,
		Priority:        0,
		DownloadEnabled: false,
		UploadEnabled:   false,
		MultiUp:         0,
		MultiDown:       0,
	}
)

func defaultTable(title string) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	if title != "" {
		t.SetTitle(title)
	}
	t.SetStyle(table.StyleColoredBright)
	return t
}

func renderRoles(roles []*store.Role, title string) {
	t := defaultTable(title)
	t.AppendHeader(table.Row{"role_id", "name", "priority", "xup", "xdn", "dl_enabled"})
	for _, role := range roles {
		t.AppendRow(table.Row{role.RoleID, role.RoleName, role.Priority, role.MultiDown,
			role.MultiUp, role.DownloadEnabled})
	}
	t.SortBy([]table.SortBy{{
		Name: "priority",
	}})
	t.Render()
}

// roleCmd represents role admin commands
var roleCmd = &cobra.Command{
	Use:               "role",
	Short:             "role commands",
	Long:              `role commands`,
	PersistentPreRunE: connectRPC,
}

// userAddCmd can be used to add users
var roleAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a role to the tracker",
	Long:  `Add a role to the tracker`,
	Run: func(cmd *cobra.Command, args []string) {
		if roleAddParam.RoleName == "" {
			log.Fatal("name cannot be empty")
		}
		client, err := client.New()
		if err != nil {
			log.Fatalf("Failed to connect to tracker")
			return
		}
		r, err2 := client.RoleAdd(context.Background(), roleAddParam)
		if err2 != nil {
			log.Fatalf("Failed to add new role: %v", err2)
		}
		role := rpc.PBToRole(r)
		renderRoles([]*store.Role{role}, "Role added successfully")
		//role.Log().Infof("Role added successfully (id: %d, name: )", r.RoleId)
	},
}

// roleListCmd can be used to list roles
var roleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List known roles",
	Long:  `List known roles`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := client.New()
		if err != nil {
			log.Fatalf("Failed to connect to tracker: %v", err)
			return
		}
		stream, err := client.RoleAll(context.Background(), &emptypb.Empty{})
		if err != nil {
			log.Fatalf("Failed to fetch roles: %v", err)
			return
		}
		var roles []*store.Role
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// read done.
				break
			}
			if err != nil {
				log.Fatalf("Failed to receive a note : %v", err)
			}
			roles = append(roles, rpc.PBToRole(in))
		}
		renderRoles(roles, "List of all roles")
	},
}

// roleDeleteCmd can be used to delete roles
var roleDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a role from the tracker",
	Long:  `Delete a role from the tracker`,
	Run: func(cmd *cobra.Command, args []string) {
		if roleDelParam.RoleId <= 0 && roleDelParam.RoleName == "" {
			log.Fatalf("Must supply one of: role name, role id")
			return
		}
		rid := &pb.RoleID{}
		if roleDelParam.RoleName != "" {
			rid.RoleName = roleDelParam.RoleName
		} else {
			rid.RoleId = roleDelParam.RoleId
		}
		client, err := client.New()
		if err != nil {
			log.Fatalf("Failed to connect to tracker")
			return
		}
		_, err2 := client.RoleDelete(context.Background(), rid)
		if err2 != nil {
			log.Fatalf("Failed to add new role: %v", err2)
		}
		log.Infof("Role deleted successfully")
	},
}

// roleSetCmd can be used to delete roles
var roleSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Update parameters for a role",
	Long:  `Update parameters for a role`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatalf("Expected 1 paramter, got %d", len(args))
		}

		for _, flag := range cmd.Flags().Args() {
			f := cmd.Flag(flag)
			if f.Changed {
				roleSetParams.UpdatedKeys = append(roleSetParams.UpdatedKeys, f.Name)
			}
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(roleCmd)
	roleCmd.AddCommand(roleListCmd)
	roleCmd.AddCommand(roleAddCmd)
	roleCmd.AddCommand(roleDeleteCmd)
	roleCmd.AddCommand(roleSetCmd)

	roleSetCmd.Flags().StringVarP(&roleSetParams.RoleName, "name", "n", "", "Name of the role")
	roleSetCmd.Flags().Int32VarP(&roleSetParams.Priority, "priority", "p", 0, "Role Priority")
	roleSetCmd.Flags().BoolVarP(&roleSetParams.DownloadEnabled, "download_enabled", "D", true, "Downloading enabled")
	roleSetCmd.Flags().BoolVarP(&roleSetParams.UploadEnabled, "upload_enabled", "U", true, "Uploading enabled")
	roleSetCmd.Flags().Float64VarP(&roleSetParams.MultiDown, "multi_down", "d", 1.0, "Download multiplier")
	roleSetCmd.Flags().Float64VarP(&roleSetParams.MultiUp, "multi_up", "u", 1.0, "Upload multiplier")

	roleDeleteCmd.Flags().StringVarP(&roleDelParam.RoleName, "name", "n", "", "Name of the role")
	roleDeleteCmd.Flags().Uint32VarP(&roleDelParam.RoleId, "id", "i", 0, "Role ID")

	roleAddCmd.Flags().StringVarP(&roleAddParam.RoleName, "name", "n", "", "Name of the role")
	roleAddCmd.Flags().Int32VarP(&roleAddParam.Priority, "priority", "p", 0, "Role Priority")
	roleAddCmd.Flags().BoolVarP(&roleAddParam.DownloadEnabled, "download_enabled", "D", true, "Downloading enabled")
	roleAddCmd.Flags().BoolVarP(&roleAddParam.UploadEnabled, "upload_enabled", "U", true, "Uploading enabled")
	roleAddCmd.Flags().Float64VarP(&roleAddParam.MultiDown, "multi_down", "d", 1.0, "Download multiplier")
	roleAddCmd.Flags().Float64VarP(&roleAddParam.MultiUp, "multi_up", "u", 1.0, "Upload multiplier")
}
