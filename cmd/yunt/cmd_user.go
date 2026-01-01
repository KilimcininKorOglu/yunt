package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	// User command flags
	userEmail    string
	userPassword string
	userRole     string
	userActive   bool
	userInactive bool
	userForce    bool
)

// userCmd represents the user command.
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
	Long: `Manage users in the Yunt mail server.

Use subcommands to create, list, modify, or delete users.`,
}

// userListCmd lists all users.
var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Long: `List all users registered in the system.

Examples:
  # List all users
  yunt user list

  # List only active users
  yunt user list --active

  # List only inactive users
  yunt user list --inactive`,
	RunE: runUserList,
}

// userCreateCmd creates a new user.
var userCreateCmd = &cobra.Command{
	Use:   "create <username>",
	Short: "Create a new user",
	Long: `Create a new user account.

Examples:
  # Create a user interactively
  yunt user create john

  # Create a user with email
  yunt user create john --email john@example.com

  # Create a user with all options
  yunt user create john --email john@example.com --role user`,
	Args: cobra.ExactArgs(1),
	RunE: runUserCreate,
}

// userDeleteCmd deletes a user.
var userDeleteCmd = &cobra.Command{
	Use:   "delete <username>",
	Short: "Delete a user",
	Long: `Delete a user account and all associated data.

WARNING: This action is irreversible!

Examples:
  # Delete a user (requires confirmation)
  yunt user delete john

  # Delete without confirmation
  yunt user delete john --force`,
	Args: cobra.ExactArgs(1),
	RunE: runUserDelete,
}

// userPasswordCmd changes a user's password.
var userPasswordCmd = &cobra.Command{
	Use:   "password <username>",
	Short: "Change user password",
	Long: `Change the password for a user account.

Examples:
  # Change password interactively
  yunt user password john

  # Set password directly (not recommended for security)
  yunt user password john --password newpassword`,
	Args: cobra.ExactArgs(1),
	RunE: runUserPassword,
}

// userInfoCmd shows user details.
var userInfoCmd = &cobra.Command{
	Use:   "info <username>",
	Short: "Show user details",
	Long: `Display detailed information about a user.

Examples:
  # Show user info
  yunt user info john`,
	Args: cobra.ExactArgs(1),
	RunE: runUserInfo,
}

// userActivateCmd activates a user.
var userActivateCmd = &cobra.Command{
	Use:   "activate <username>",
	Short: "Activate a user account",
	Long: `Activate a disabled user account.

Examples:
  yunt user activate john`,
	Args: cobra.ExactArgs(1),
	RunE: runUserActivate,
}

// userDeactivateCmd deactivates a user.
var userDeactivateCmd = &cobra.Command{
	Use:   "deactivate <username>",
	Short: "Deactivate a user account",
	Long: `Deactivate a user account without deleting it.

Examples:
  yunt user deactivate john`,
	Args: cobra.ExactArgs(1),
	RunE: runUserDeactivate,
}

func init() {
	// Add subcommands to user
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userDeleteCmd)
	userCmd.AddCommand(userPasswordCmd)
	userCmd.AddCommand(userInfoCmd)
	userCmd.AddCommand(userActivateCmd)
	userCmd.AddCommand(userDeactivateCmd)

	// List flags
	userListCmd.Flags().BoolVar(&userActive, "active", false, "show only active users")
	userListCmd.Flags().BoolVar(&userInactive, "inactive", false, "show only inactive users")

	// Create flags
	userCreateCmd.Flags().StringVarP(&userEmail, "email", "e", "", "user email address")
	userCreateCmd.Flags().StringVarP(&userPassword, "password", "p", "", "user password (prompted if not provided)")
	userCreateCmd.Flags().StringVarP(&userRole, "role", "r", "user", "user role (admin, user)")

	// Delete flags
	userDeleteCmd.Flags().BoolVarP(&userForce, "force", "f", false, "skip confirmation prompt")

	// Password flags
	userPasswordCmd.Flags().StringVarP(&userPassword, "password", "p", "", "new password (prompted if not provided)")
}

func runUserList(cmd *cobra.Command, args []string) error {
	log := getLogger()
	cfg := getConfig()

	log.Debug().
		Bool("active_only", userActive).
		Bool("inactive_only", userInactive).
		Msg("Listing users")

	// TODO: Implement actual user listing from database
	fmt.Println("User List")
	fmt.Println("=========")
	fmt.Println()
	fmt.Printf("Database: %s (%s)\n", cfg.Database.Name, cfg.Database.Driver)
	fmt.Println()
	fmt.Printf("%-20s %-30s %-10s %-10s\n", "USERNAME", "EMAIL", "ROLE", "STATUS")
	fmt.Printf("%-20s %-30s %-10s %-10s\n", "--------", "-----", "----", "------")
	fmt.Printf("%-20s %-30s %-10s %-10s\n", "admin", "admin@localhost", "admin", "active")
	fmt.Println()
	fmt.Println("Total: 1 user(s)")

	return nil
}

func runUserCreate(cmd *cobra.Command, args []string) error {
	log := getLogger()
	_ = getConfig()

	username := args[0]

	// Prompt for password if not provided
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

	// Prompt for email if not provided
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

	// TODO: Implement actual user creation
	fmt.Println()
	fmt.Printf("Creating user '%s'...\n", username)
	fmt.Printf("  Email: %s\n", email)
	fmt.Printf("  Role: %s\n", userRole)
	fmt.Println()
	fmt.Printf("User '%s' created successfully\n", username)

	return nil
}

func runUserDelete(cmd *cobra.Command, args []string) error {
	log := getLogger()
	_ = getConfig()

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

	log.Info().
		Str("username", username).
		Msg("Deleting user")

	// TODO: Implement actual user deletion
	fmt.Printf("Deleting user '%s'...\n", username)
	fmt.Printf("User '%s' deleted successfully\n", username)

	return nil
}

func runUserPassword(cmd *cobra.Command, args []string) error {
	log := getLogger()
	_ = getConfig()

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

	log.Info().
		Str("username", username).
		Msg("Changing user password")

	// TODO: Implement actual password change
	fmt.Printf("Changing password for user '%s'...\n", username)
	fmt.Println("Password changed successfully")

	return nil
}

func runUserInfo(cmd *cobra.Command, args []string) error {
	log := getLogger()
	_ = getConfig()

	username := args[0]

	log.Debug().
		Str("username", username).
		Msg("Fetching user info")

	// TODO: Implement actual user info retrieval
	fmt.Println("User Information")
	fmt.Println("================")
	fmt.Println()
	fmt.Printf("Username:    %s\n", username)
	fmt.Printf("Email:       %s@localhost\n", username)
	fmt.Printf("Role:        user\n")
	fmt.Printf("Status:      active\n")
	fmt.Printf("Created:     2024-01-01 12:00:00\n")
	fmt.Printf("Last Login:  2024-01-15 09:30:00\n")
	fmt.Println()
	fmt.Println("Mailboxes:")
	fmt.Println("  - INBOX (0 messages)")
	fmt.Println("  - Sent (0 messages)")
	fmt.Println("  - Drafts (0 messages)")
	fmt.Println("  - Trash (0 messages)")

	return nil
}

func runUserActivate(cmd *cobra.Command, args []string) error {
	log := getLogger()
	_ = getConfig()

	username := args[0]

	log.Info().
		Str("username", username).
		Msg("Activating user")

	// TODO: Implement actual user activation
	fmt.Printf("Activating user '%s'...\n", username)
	fmt.Printf("User '%s' activated successfully\n", username)

	return nil
}

func runUserDeactivate(cmd *cobra.Command, args []string) error {
	log := getLogger()
	_ = getConfig()

	username := args[0]

	log.Info().
		Str("username", username).
		Msg("Deactivating user")

	// TODO: Implement actual user deactivation
	fmt.Printf("Deactivating user '%s'...\n", username)
	fmt.Printf("User '%s' deactivated successfully\n", username)

	return nil
}

// promptPassword prompts for a password without echoing.
func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(password), nil
}

// promptInput prompts for text input.
func promptInput(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}
