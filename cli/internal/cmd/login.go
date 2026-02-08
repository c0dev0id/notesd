package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/c0dev0id/notesd/cli/internal/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with a notesd server",
	RunE:  runLogin,
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Create a new account on a notesd server",
	RunE:  runRegister,
}

func init() {
	loginCmd.Flags().StringP("server", "s", "", "Server URL (e.g. http://localhost:8080)")
	loginCmd.Flags().StringP("email", "e", "", "Email address")
	loginCmd.Flags().StringP("password", "p", "", "Password (omit to prompt)")
	loginCmd.Flags().StringP("device", "d", "", "Device ID (default: hostname)")

	registerCmd.Flags().StringP("server", "s", "", "Server URL")
	registerCmd.Flags().StringP("email", "e", "", "Email address")
	registerCmd.Flags().StringP("password", "p", "", "Password (omit to prompt)")
	registerCmd.Flags().StringP("name", "n", "", "Display name")
}

func runLogin(cmd *cobra.Command, args []string) error {
	var err error
	cl, err = client.New()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	serverURL, _ := cmd.Flags().GetString("server")
	if serverURL == "" {
		defaultURL := cl.BaseURL
		if defaultURL == "" {
			defaultURL = "http://localhost:8080"
		}
		serverURL = prompt(reader, fmt.Sprintf("Server URL [%s]: ", defaultURL))
		if serverURL == "" {
			serverURL = defaultURL
		}
	}
	serverURL = strings.TrimRight(serverURL, "/")

	email, _ := cmd.Flags().GetString("email")
	if email == "" {
		email = prompt(reader, "Email: ")
	}

	password, _ := cmd.Flags().GetString("password")
	if password == "" {
		password = promptPassword("Password: ")
	}

	deviceID, _ := cmd.Flags().GetString("device")
	if deviceID == "" {
		deviceID = cl.DeviceID
	}

	if err := cl.Login(serverURL, email, password, deviceID); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	s := cl.SessionInfo()
	fmt.Printf("Logged in as %s (%s)\n", s.DisplayName, s.Email)
	return nil
}

func runRegister(cmd *cobra.Command, args []string) error {
	var err error
	cl, err = client.New()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	serverURL, _ := cmd.Flags().GetString("server")
	if serverURL == "" {
		defaultURL := cl.BaseURL
		if defaultURL == "" {
			defaultURL = "http://localhost:8080"
		}
		serverURL = prompt(reader, fmt.Sprintf("Server URL [%s]: ", defaultURL))
		if serverURL == "" {
			serverURL = defaultURL
		}
	}
	serverURL = strings.TrimRight(serverURL, "/")

	email, _ := cmd.Flags().GetString("email")
	if email == "" {
		email = prompt(reader, "Email: ")
	}

	displayName, _ := cmd.Flags().GetString("name")
	if displayName == "" {
		displayName = prompt(reader, "Display name: ")
	}

	password, _ := cmd.Flags().GetString("password")
	if password == "" {
		password = promptPassword("Password: ")
		confirm := promptPassword("Confirm password: ")
		if password != confirm {
			return fmt.Errorf("passwords do not match")
		}
	}

	if err := cl.Register(serverURL, email, password, displayName); err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	fmt.Println("Account created. You can now log in with: notesd login")
	return nil
}

func prompt(reader *bufio.Reader, label string) string {
	fmt.Fprint(os.Stderr, label)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptPassword(label string) string {
	fmt.Fprint(os.Stderr, label)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return ""
		}
		return string(b)
	}
	// Fallback for piped input
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	fmt.Fprintln(os.Stderr)
	return strings.TrimSpace(line)
}
