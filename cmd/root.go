package cmd

import (
	"fmt"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
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
	flagDebugLog            = "debug"
	flagDebugLogShort       = "d"
	envOCMToken             = "OCM_TOKEN"
)

var rootCmd = &cobra.Command{
	Use:   "ocm-exporter",
	Short: "Starts the OCM metrics exporter",
	RunE: func(cmd *cobra.Command, args []string) error {

		var (
			ocmToken   string
			port       string
			orgID      string
			debug      bool
			logger     *logging.GoLogger
			connection *sdk.Connection
			err        error
		)

		if ocmToken, err = getOCMTokenFlag(cmd); err != nil {
			return err
		}
		if port, err = getPortFlag(cmd); err != nil {
			return err
		}
		if orgID, err = getOrgIDFlag(cmd); err != nil {
			return err
		}
		if debug, err = getDebugFlag(cmd); err != nil {
			return err
		}
		if logger, err = sdk.NewGoLoggerBuilder().Debug(debug).Build(); err != nil {
			return fmt.Errorf("failed to create logger: %v", err)
		}
		if connection, err = sdk.NewConnectionBuilder().Logger(logger).Tokens(ocmToken).Build(); err != nil {
			return fmt.Errorf("failed to create ocm connection: %v", err)
		}
		defer connection.Close()

		if orgID, err = getOrgID(orgID, connection); err != nil {
			return err
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

func getOrgID(orgID string, connection *sdk.Connection) (string, error) {
	if orgID == "" {
		// Get organization of current user:
		userConn, err := connection.AccountsMgmt().V1().CurrentAccount().Get().Send()
		if err != nil {
			return "", fmt.Errorf("failed to retrieve current user information: %v", err)
		}
		userOrg, _ := userConn.Body().GetOrganization()
		orgID = userOrg.ID()
	}
	return orgID, nil
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

	rootCmd.Flags().BoolP(flagDebugLog, flagDebugLogShort, false, "Enable debug logging")
}

func getOCMTokenFlag(cmd *cobra.Command) (string, error) {
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

func getPortFlag(cmd *cobra.Command) (string, error) {
	port, err := cmd.Flags().GetString(flagPort)
	if err != nil {
		return "", fmt.Errorf("failed to get %s flag: %v", flagPort, err)
	}
	if port == "" {
		port = defaultPort
	}
	return port, nil
}

func getOrgIDFlag(cmd *cobra.Command) (string, error) {
	orgID, err := cmd.Flags().GetString(flagOrganizationID)
	if err != nil {
		return "", fmt.Errorf("failed to get %s flag: %v", flagOrganizationID, err)
	}
	return orgID, nil
}

func getDebugFlag(cmd *cobra.Command) (bool, error) {
	debug, err := cmd.Flags().GetBool(flagDebugLog)
	if err != nil {
		return false, fmt.Errorf("failed to get %s flag: %v", flagDebugLog, err)
	}
	return debug, nil
}
