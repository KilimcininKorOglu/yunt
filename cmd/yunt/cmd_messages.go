package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

var (
	messagesLimit  int
	messagesOffset int
	messagesFrom   string
	messagesTo     string
	messagesFormat string
	messagesForce  bool
)

var messagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "Manage messages",
	Long: `Manage messages in the Yunt mail server.

Use subcommands to list, view, delete, or export messages.`,
}

var messagesListCmd = &cobra.Command{
	Use:   "list [mailbox]",
	Short: "List messages",
	Long: `List messages in a mailbox.

Examples:
  yunt messages list
  yunt messages list --limit 50 --from user@example.com`,
	RunE: runMessagesList,
}

var messagesViewCmd = &cobra.Command{
	Use:   "view <message-id>",
	Short: "View a message",
	Long: `View the contents of a specific message.

Examples:
  yunt messages view msg-123
  yunt messages view msg-123 --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runMessagesView,
}

var messagesDeleteCmd = &cobra.Command{
	Use:   "delete <message-id> [message-id...]",
	Short: "Delete messages",
	Long: `Delete one or more messages.

Examples:
  yunt messages delete msg-123
  yunt messages delete msg-123 msg-456 --force`,
	Args: cobra.MinimumNArgs(1),
	RunE: runMessagesDelete,
}

var messagesPurgeCmd = &cobra.Command{
	Use:     "purge [mailbox]",
	Aliases: []string{"delete-all"},
	Short:   "Purge all messages",
	Long: `Delete all messages in a mailbox or all mailboxes.

Examples:
  yunt messages purge INBOX --force
  yunt messages purge --force`,
	RunE: runMessagesPurge,
}

var messagesExportCmd = &cobra.Command{
	Use:   "export <output-file>",
	Short: "Export messages to JSON file",
	Long: `Export messages to a JSON file.

Examples:
  yunt messages export backup.json`,
	Args: cobra.ExactArgs(1),
	RunE: runMessagesExport,
}

var messagesStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show message statistics",
	RunE:  runMessagesStats,
}

func init() {
	messagesCmd.AddCommand(messagesListCmd)
	messagesCmd.AddCommand(messagesViewCmd)
	messagesCmd.AddCommand(messagesDeleteCmd)
	messagesCmd.AddCommand(messagesPurgeCmd)
	messagesCmd.AddCommand(messagesExportCmd)
	messagesCmd.AddCommand(messagesStatsCmd)

	messagesListCmd.Flags().IntVarP(&messagesLimit, "limit", "l", 25, "maximum number of messages to display")
	messagesListCmd.Flags().IntVarP(&messagesOffset, "offset", "o", 0, "offset for pagination")
	messagesListCmd.Flags().StringVar(&messagesFrom, "from", "", "filter by sender address")
	messagesListCmd.Flags().StringVar(&messagesTo, "to", "", "filter by recipient address")

	messagesViewCmd.Flags().StringVarP(&messagesFormat, "format", "f", "text", "output format (text, raw, json)")

	messagesDeleteCmd.Flags().BoolVar(&messagesForce, "force", false, "skip confirmation prompt")

	messagesPurgeCmd.Flags().BoolVar(&messagesForce, "force", false, "confirm purge operation (required)")

	messagesExportCmd.Flags().StringVarP(&messagesFormat, "format", "f", "json", "export format (json)")
	messagesExportCmd.Flags().StringVar(&messagesFrom, "from", "", "source mailbox name")
}

func runMessagesList(cmd *cobra.Command, args []string) error {
	repo, err := initRepo()
	if err != nil {
		return err
	}
	defer repo.Close()

	ctx := context.Background()

	filter := &repository.MessageFilter{}
	if messagesFrom != "" {
		filter.FromAddress = messagesFrom
	}

	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{
			Page:    (messagesOffset / messagesLimit) + 1,
			PerPage: messagesLimit,
		},
	}

	result, err := repo.Messages().List(ctx, filter, opts)
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	fmt.Printf("%-36s %-25s %-40s %s\n", "ID", "FROM", "SUBJECT", "DATE")
	fmt.Printf("%-36s %-25s %-40s %s\n", strings.Repeat("-", 36), strings.Repeat("-", 25), strings.Repeat("-", 40), strings.Repeat("-", 19))

	for _, msg := range result.Items {
		from := msg.From.Address
		if len(from) > 24 {
			from = from[:21] + "..."
		}
		subject := msg.Subject
		if len(subject) > 39 {
			subject = subject[:36] + "..."
		}
		date := msg.ReceivedAt.Time.Format("2006-01-02 15:04:05")
		fmt.Printf("%-36s %-25s %-40s %s\n", msg.ID, from, subject, date)
	}

	fmt.Printf("\nShowing %d of %d message(s)\n", len(result.Items), result.Total)
	return nil
}

func runMessagesView(cmd *cobra.Command, args []string) error {
	messageID := domain.ID(args[0])

	repo, err := initRepo()
	if err != nil {
		return err
	}
	defer repo.Close()

	ctx := context.Background()
	msg, err := repo.Messages().GetByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("message not found: %w", err)
	}

	switch messagesFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(msg)
	case "raw":
		fmt.Print(string(msg.RawBody))
		return nil
	default:
		fmt.Printf("From:    %s <%s>\n", msg.From.Name, msg.From.Address)
		fmt.Printf("Subject: %s\n", msg.Subject)
		fmt.Printf("Date:    %s\n", msg.ReceivedAt.Time.Format("2006-01-02 15:04:05"))
		fmt.Printf("Status:  %s\n", msg.Status)
		fmt.Println(strings.Repeat("-", 60))
		if msg.TextBody != "" {
			fmt.Println(msg.TextBody)
		} else {
			fmt.Println("(no text body)")
		}
		return nil
	}
}

func runMessagesDelete(cmd *cobra.Command, args []string) error {
	if !messagesForce {
		fmt.Printf("Delete %d message(s)? Type 'yes' to confirm: ", len(args))
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(strings.TrimSpace(confirm)) != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	repo, err := initRepo()
	if err != nil {
		return err
	}
	defer repo.Close()

	ctx := context.Background()
	deleted, failed := 0, 0

	for _, id := range args {
		if err := repo.Messages().Delete(ctx, domain.ID(id)); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete %s: %v\n", id, err)
			failed++
		} else {
			deleted++
		}
	}

	fmt.Printf("Deleted: %d, Failed: %d\n", deleted, failed)
	if failed > 0 {
		return fmt.Errorf("%d deletion(s) failed", failed)
	}
	return nil
}

func runMessagesPurge(cmd *cobra.Command, args []string) error {
	if !messagesForce {
		return fmt.Errorf("purge requires --force flag to confirm")
	}

	repo, err := initRepo()
	if err != nil {
		return err
	}
	defer repo.Close()

	ctx := context.Background()

	if len(args) > 0 {
		mailboxName := args[0]
		mailboxes, err := repo.Mailboxes().List(ctx, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to list mailboxes: %w", err)
		}

		for _, mb := range mailboxes.Items {
			if strings.EqualFold(mb.Name, mailboxName) {
				count, err := repo.Messages().DeleteByMailbox(ctx, mb.ID)
				if err != nil {
					return fmt.Errorf("failed to purge mailbox: %w", err)
				}
				fmt.Printf("Purged %d message(s) from '%s'\n", count, mailboxName)
				return nil
			}
		}
		return fmt.Errorf("mailbox '%s' not found", mailboxName)
	}

	mailboxes, err := repo.Mailboxes().List(ctx, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to list mailboxes: %w", err)
	}

	total := int64(0)
	for _, mb := range mailboxes.Items {
		count, err := repo.Messages().DeleteByMailbox(ctx, mb.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to purge '%s': %v\n", mb.Name, err)
			continue
		}
		total += count
	}

	fmt.Printf("Purged %d message(s) from all mailboxes\n", total)
	return nil
}

func runMessagesExport(cmd *cobra.Command, args []string) error {
	outputFile := args[0]

	repo, err := initRepo()
	if err != nil {
		return err
	}
	defer repo.Close()

	ctx := context.Background()

	result, err := repo.Messages().List(ctx, nil, &repository.ListOptions{
		Pagination: &repository.PaginationOptions{Page: 1, PerPage: 10000},
	})
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result.Items); err != nil {
		return fmt.Errorf("failed to write messages: %w", err)
	}

	fmt.Printf("Exported %d message(s) to %s\n", len(result.Items), outputFile)
	return nil
}

func runMessagesStats(cmd *cobra.Command, args []string) error {
	repo, err := initRepo()
	if err != nil {
		return err
	}
	defer repo.Close()

	ctx := context.Background()

	totalMsgs, err := repo.Messages().Count(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to count messages: %w", err)
	}

	mailboxes, err := repo.Mailboxes().List(ctx, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to list mailboxes: %w", err)
	}

	fmt.Println("Message Statistics")
	fmt.Println("==================")
	fmt.Printf("Total messages: %d\n", totalMsgs)
	fmt.Printf("Mailboxes:      %d\n\n", len(mailboxes.Items))

	if len(mailboxes.Items) > 0 {
		fmt.Printf("%-20s %10s %10s\n", "MAILBOX", "MESSAGES", "UNREAD")
		fmt.Printf("%-20s %10s %10s\n", "-------", "--------", "------")
		for _, mb := range mailboxes.Items {
			fmt.Printf("%-20s %10d %10d\n", mb.Name, mb.MessageCount, mb.UnreadCount)
		}
	}

	return nil
}
