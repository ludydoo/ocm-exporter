package cmd

import (
	"fmt"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

const (
	defaultPort             = "9090"
	flagPort                = "port"
	flagPortShort           = "p"
	flagOrganizationID      = "organization-id"
	flatOrganizationIDShort = "o"
	flagOCMTokenPath        = "ocm-token-path"
	flagOCMTokenPathShort   = "t"
	envOCMToken             = "OCM_TOKEN"
)

var rootCmd = &cobra.Command{
	Use:   "ocm-exporter",
	Short: "Starts the OCM metrics exporter",
	RunE: func(cmd *cobra.Command, args []string) error {
		ocmToken, err := getOCMToken(cmd)
		if err != nil {
			return err
		}

		port, err := getPort(cmd)
		if err != nil {
			return err
		}

		orgID, err := getOrgID(cmd)
		if err != nil {
			return err
		}

		logger, err := sdk.NewGoLoggerBuilder().
			Debug(true).
			Build()

		if err != nil {
			return fmt.Errorf("failed to create logger: %v", err)
		}

		connection, err := sdk.NewConnectionBuilder().
			Logger(logger).
			Tokens(ocmToken).
			Build()
		if err != nil {
			return fmt.Errorf("failed to create ocm connection: %v", err)
		}
		defer connection.Close()

		if orgID == "" {
			// Get organization of current user:
			userConn, err := connection.AccountsMgmt().V1().CurrentAccount().Get().Send()
			if err != nil {
				return fmt.Errorf("failed to retrieve current user information: %v", err)
			}
			userOrg, _ := userConn.Body().GetOrganization()
			orgID = userOrg.ID()
		}

		orgCollection := connection.AccountsMgmt().V1().Organizations().Organization(orgID)
		if err != nil {
			return fmt.Errorf("failed to retrieve organization information: %v", err)
		}

		quotaClient := orgCollection.QuotaCost()

		collector := newOcmCollector(quotaClient)
		prometheus.MustRegister(collector)

		http.Handle("/metrics", promhttp.Handler())
		return http.ListenAndServe(":"+port, nil)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	rootCmd.Flags().StringP(flagPort, flagPortShort, defaultPort, "Port to listen on")

	rootCmd.Flags().StringP(flagOrganizationID, flatOrganizationIDShort, "", "Organization ID to query quotas for")

	rootCmd.Flags().StringP(flagOCMTokenPath, flagOCMTokenPathShort, "", "Path to file containing OCM token")
}

func getOCMToken(cmd *cobra.Command) (string, error) {
	ocmToken := os.Getenv(envOCMToken)
	ocmTokenPath, err := cmd.Flags().GetString(flagOCMTokenPath)
	if err != nil {
		return "", fmt.Errorf("failed to get %s flag: %v", flagOCMTokenPath, err)
	}
	if ocmTokenPath != "" {
		ocmTokenBytes, err := os.ReadFile(ocmTokenPath)
		if err != nil {
			return "", fmt.Errorf("failed to read OCM token file: %v", err)
		}
		return string(ocmTokenBytes), nil
	}
	return ocmToken, nil
}

func getPort(cmd *cobra.Command) (string, error) {
	port, err := cmd.Flags().GetString(flagPort)
	if err != nil {
		return "", fmt.Errorf("failed to get %s flag: %v", flagPort, err)
	}
	if port == "" {
		port = defaultPort
	}
	return port, nil
}

func getOrgID(cmd *cobra.Command) (string, error) {
	orgID, err := cmd.Flags().GetString(flagOrganizationID)
	if err != nil {
		return "", fmt.Errorf("failed to get %s flag: %v", flagOrganizationID, err)
	}
	return orgID, nil
}
