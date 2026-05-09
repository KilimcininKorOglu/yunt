package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	"yunt/internal/domain"
	"yunt/internal/repository"
	"yunt/internal/service"
)

var (
	userEmail    string
	userPassword string
	userRole     string
	userActive   bool
	userInactive bool
	userForce    bool
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
	Long: `Manage users in the Yunt mail server.

Use subcommands to create, list, modify, or delete users.`,
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Long: `List all users registered in the system.

Examples:
  yunt user list
  yunt user list --active
  yunt user list --inactive`,
	RunE: runUserList,
}

var userCreateCmd = &cobra.Command{
	Use:   "create <username>",
	Short: "Create a new user",
	Long: `Create a new user account.

Examples:
  yunt user create john
  yunt user create john --email john@example.com
  yunt user create john --email john@example.com --role admin`,
	Args: cobra.ExactArgs(1),
	RunE: runUserCreate,
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete <username>",
	Short: "Delete a user",
	Long: `Delete a user account.

Examples:
  yunt user delete john
  yunt user delete john --force`,
	Args: cobra.ExactArgs(1),
	RunE: runUserDelete,
}

var userPasswordCmd = &cobra.Command{
	Use:   "password <username>",
	Short: "Change user password",
	Long: `Change the password for a user account.

Examples:
  yunt user password john
  yunt user password john --password newpassword`,
	Args: cobra.ExactArgs(1),
	RunE: runUserPassword,
}

var userInfoCmd = &cobra.Command{
	Use:   "info <username>",
	Short: "Show user details",
	Long: `Display detailed information about a user.

Examples:
  yunt user info john`,
	Args: cobra.ExactArgs(1),
	RunE: runUserInfo,
}

var userActivateCmd = &cobra.Command{
	Use:   "activate <username>",
	Short: "Activate a user account",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserActivate,
}

var userDeactivateCmd = &cobra.Command{
	Use:   "deactivate <username>",
	Short: "Deactivate a user account",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserDeactivate,
}

func init() {
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userDeleteCmd)
	userCmd.AddCommand(userPasswordCmd)
	userCmd.AddCommand(userInfoCmd)
	userCmd.AddCommand(userActivateCmd)
	userCmd.AddCommand(userDeactivateCmd)

	userListCmd.Flags().BoolVar(&userActive, "active", false, "show only active users")
	userListCmd.Flags().BoolVar(&userInactive, "inactive", false, "show only inactive users")

	userCreateCmd.Flags().StringVarP(&userEmail, "email", "e", "", "user email address")
	userCreateCmd.Flags().StringVarP(&userPassword, "password", "p", "", "user password (prompted if not provided)")
	userCreateCmd.Flags().StringVarP(&userRole, "role", "r", "user", "user role (admin, user)")

	userDeleteCmd.Flags().BoolVarP(&userForce, "force", "f", false, "skip confirmation prompt")

	userPasswordCmd.Flags().StringVarP(&userPassword, "password", "p", "", "new password (prompted if not provided)")
}

func runUserList(cmd *cobra.Command, args []string) error {
	log := getLogger()

	log.Debug().
		Bool("active_only", userActive).
		Bool("inactive_only", userInactive).
		Msg("Listing users")

	repo, err := initRepo()
	if err != nil {
		return err
	}
	defer repo.Close()

	ctx := context.Background()

	var filter *repository.UserFilter
	if userActive || userInactive {
		filter = &repository.UserFilter{}
		if userActive {
			s := domain.StatusActive
			filter.Status = &s
		} else if userInactive {
			s := domain.StatusInactive
			filter.Status = &s
		}
	}

	result, err := repo.Users().List(ctx, filter, &repository.ListOptions{
		Pagination: &repository.PaginationOptions{Page: 1, PerPage: 100},
	})
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	fmt.Printf("%-20s %-30s %-10s %-10s\n", "USERNAME", "EMAIL", "ROLE", "STATUS")
	fmt.Printf("%-20s %-30s %-10s %-10s\n", "--------", "-----", "----", "------")
	for _, u := range result.Items {
		fmt.Printf("%-20s %-30s %-10s %-10s\n", u.Username, u.Email, u.Role, u.Status)
	}
	fmt.Printf("\nTotal: %d user(s)\n", result.Total)

	return nil
}

func runUserCreate(cmd *cobra.Command, args []string) error {
	log := getLogger()
	cfg := getConfig()

	username := args[0]

	password := userPassword
	if password == "" {
		var err error
		password, err = promptPassword("Enter password: ")
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		confirm, err := promptPassword("Confirm password: ")
		if err != nil {
			return fmt.Errorf("failed to read password confirmation: %w", err)
		}
		if password != confirm {
			return fmt.Errorf("passwords do not match")
		}
	}

	email := userEmail
	if email == "" {
		var err error
		email, err = promptInput("Enter email: ")
		if err != nil {
			return fmt.Errorf("failed to read email: %w", err)
		}
	}

	log.Info().
		Str("username", username).
		Str("email", email).
		Str("role", userRole).
		Msg("Creating user")

	repo, err := initRepo()
	if err != nil {
		return err
	}
	defer repo.Close()

	userSvc := service.NewUserService(cfg.Auth, repo.Users())
	ctx := context.Background()

	input := &domain.UserCreateInput{
		Username: username,
		Email:    email,
		Password: password,
		Role:     domain.UserRole(userRole),
	}

	user, err := userSvc.Create(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("User '%s' created successfully (ID: %s)\n", user.Username, user.ID)
	return nil
}

func runUserDelete(cmd *cobra.Command, args []string) error {
	log := getLogger()
	username := args[0]

	if !userForce {
		fmt.Printf("Are you sure you want to delete user '%s'? This cannot be undone.\n", username)
		confirm, err := promptInput("Type 'yes' to confirm: ")
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if strings.ToLower(strings.TrimSpace(confirm)) != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	log.Info().Str("username", username).Msg("Deleting user")

	repo, err := initRepo()
	if err != nil {
		return err
	}
	defer repo.Close()

	ctx := context.Background()
	user, err := repo.Users().GetByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("user '%s' not found: %w", username, err)
	}

	if err := repo.Users().SoftDelete(ctx, user.ID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	fmt.Printf("User '%s' deleted successfully\n", username)
	return nil
}

func runUserPassword(cmd *cobra.Command, args []string) error {
	log := getLogger()
	cfg := getConfig()
	username := args[0]

	password := userPassword
	if password == "" {
		var err error
		password, err = promptPassword("Enter new password: ")
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		confirm, err := promptPassword("Confirm new password: ")
		if err != nil {
			return fmt.Errorf("failed to read password confirmation: %w", err)
		}
		if password != confirm {
			return fmt.Errorf("passwords do not match")
		}
	}

	log.Info().Str("username", username).Msg("Changing user password")

	repo, err := initRepo()
	if err != nil {
		return err
	}
	defer repo.Close()

	ctx := context.Background()
	user, err := repo.Users().GetByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("user '%s' not found: %w", username, err)
	}

	userSvc := service.NewUserService(cfg.Auth, repo.Users())
	if err := userSvc.UpdatePassword(ctx, user.ID, password); err != nil {
		return fmt.Errorf("failed to change password: %w", err)
	}

	fmt.Println("Password changed successfully")
	return nil
}

func runUserInfo(cmd *cobra.Command, args []string) error {
	username := args[0]

	repo, err := initRepo()
	if err != nil {
		return err
	}
	defer repo.Close()

	ctx := context.Background()
	user, err := repo.Users().GetByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("user '%s' not found: %w", username, err)
	}

	fmt.Println("User Information")
	fmt.Println("================")
	fmt.Printf("ID:          %s\n", user.ID)
	fmt.Printf("Username:    %s\n", user.Username)
	fmt.Printf("Email:       %s\n", user.Email)
	fmt.Printf("Display:     %s\n", user.DisplayName)
	fmt.Printf("Role:        %s\n", user.Role)
	fmt.Printf("Status:      %s\n", user.Status)
	fmt.Printf("Created:     %s\n", user.CreatedAt.Time.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", user.UpdatedAt.Time.Format("2006-01-02 15:04:05"))
	if user.LastLoginAt != nil {
		fmt.Printf("Last Login:  %s\n", user.LastLoginAt.Time.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("Last Login:  never\n")
	}

	return nil
}

func runUserActivate(cmd *cobra.Command, args []string) error {
	log := getLogger()
	username := args[0]

	log.Info().Str("username", username).Msg("Activating user")

	repo, err := initRepo()
	if err != nil {
		return err
	}
	defer repo.Close()

	ctx := context.Background()
	user, err := repo.Users().GetByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("user '%s' not found: %w", username, err)
	}

	if err := repo.Users().UpdateStatus(ctx, user.ID, domain.StatusActive); err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}

	fmt.Printf("User '%s' activated successfully\n", username)
	return nil
}

func runUserDeactivate(cmd *cobra.Command, args []string) error {
	log := getLogger()
	username := args[0]

	log.Info().Str("username", username).Msg("Deactivating user")

	repo, err := initRepo()
	if err != nil {
		return err
	}
	defer repo.Close()

	ctx := context.Background()
	user, err := repo.Users().GetByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("user '%s' not found: %w", username, err)
	}

	if err := repo.Users().UpdateStatus(ctx, user.ID, domain.StatusInactive); err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	fmt.Printf("User '%s' deactivated successfully\n", username)
	return nil
}

func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(password), nil
}

func promptInput(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}
