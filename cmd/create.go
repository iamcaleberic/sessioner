package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/spf13/cobra"
)

// Verbose Flag value
var Verbose bool

// MFASerial Flag value
var MFASerial string

// SubProfiles Flag value
var SubProfiles map[string]string

// SessionRegion Flag value
var SessionRegion string

// SessionSource Flag value
var SessionSource string

// SessionOutput Flag value
var SessionOutput string

// SessionProfile Flag value
var SessionProfile string

// SessionDuration Flag value
var SessionDuration int

func init() {
	homeDir, err := os.UserHomeDir()
	_ = err
	configPath := filepath.Join(homeDir, "/.aws/config-sessioner")
	credentialsPath := filepath.Join(homeDir, "/.aws/credentials")

	awsCmd.Flags().StringVarP(&SessionProfile, "session-profile", "p", "", "Session Profile within for your AWS root credentials to load for session config")
	awsCmd.Flags().IntVarP(&SessionDuration, "session-duration", "d", 1, "Expiry duration of the STS credentials in hours.")
	awsCmd.Flags().StringVarP(&SessionSource, "session-source", "s", credentialsPath, "Source path for your root credentials")
	awsCmd.Flags().StringVarP(&SessionOutput, "session-output", "o", configPath, "Path to output sts credentials")
	awsCmd.Flags().StringVarP(&SessionRegion, "session-region", "r", "eu-west-1", "AWS region of the STS credentials.")
	awsCmd.Flags().StringVarP(&MFASerial, "mfa-serial", "m", "", "AWS MFA token serial")
	awsCmd.Flags().StringToStringVarP(&SubProfiles, "sub-profiles", "c", make(map[string]string), "Profiles that source the sts credentials")

	awsCmd.MarkFlagRequired("mfa-serial")
	awsCmd.MarkFlagRequired("session-profile")

	createCmd.AddCommand(awsCmd)
	rootCmd.AddCommand(createCmd)
}

// Create a new session
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new session",
	Long:  `Create a new session and append the config`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Print: " + strings.Join(args, " "))
	},
}

// Create new aws session
var awsCmd = &cobra.Command{
	Use:   "aws",
	Short: "Create a new AWS session",
	Long:  `Create a new AWS session and append the config to your path DEFAULT=$HOME/.aws/config-sessioner`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Print: " + strings.Join(args, " "))
		fmt.Println(time.Duration(SessionDuration).Minutes())

		sess, err := session.NewSessionWithOptions(session.Options{
			// Specify profile to load for the session's config
			Profile: SessionProfile,
			// Provide SDK Config options, such as Region.
			Config: aws.Config{
				Region: aws.String(SessionRegion),
			},
			// Force enable Shared Config support
			// SharedConfigState: session.SharedConfigEnable,
		})

		if err != nil {
			fmt.Println(err)
		}

		for profile, arn := range SubProfiles {
			fmt.Println(arn)

			creds := stscreds.NewCredentials(sess, arn, func(p *stscreds.AssumeRoleProvider) {
				p.SerialNumber = aws.String(MFASerial)
				p.TokenProvider = stscreds.StdinTokenProvider
				p.Duration = toDuration(SessionDuration)
			})

			// Create service client value configured for credentials
			// from assumed role.
			credsValue, err := creds.Get()
			if err != nil {
				fmt.Println("Error getting session credentials", err)
			}
			fmt.Println("Creds:", credsValue.AccessKeyID)
			stsProfilename := profile
			stsConfigValues := fmt.Sprintf("\n[profile %v]\naws_access_key_id=%v\naws_secret_access_key=%v\naws_session_token=%v\n",
				stsProfilename, credsValue.AccessKeyID, credsValue.SecretAccessKey, credsValue.SessionToken)

			f, err := os.OpenFile(SessionOutput, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			check(err)

			defer f.Close()

			_, err = f.WriteString(stsConfigValues)
			f.Sync()
		}

	},
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func toDuration(hours int) time.Duration {
	s := strconv.Itoa(hours)
	d, err := time.ParseDuration(s + "h")
	if err != nil {
		fmt.Println(err)
	}
	return d
}
