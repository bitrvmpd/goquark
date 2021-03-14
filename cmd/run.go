package cmd

import (
	"log"

	usbUtils "github.com/bitrvmpd/goquark/internal/pkg/usb"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Starts Goldleaf client",
	Long: `Starts listening for Goldleaf connection and serves the specified folders.
	If no folders are specified it serves the current one`,
	Run: func(cmd *cobra.Command, args []string) {
		//quarkVersion := "0.4.0"
		//minGoldleafVersion := "0.8.0"
		c := usbUtils.NewCommand()

		// Reads goldleaf description
		s, err := c.RetrieveDesc()
		if err != nil {
			log.Fatalf("ERROR: %v", err)
		}

		log.Printf("DESC: %v", s)

		// Reads goldleaf's version number
		s, err = c.RetrieveSerialNumber()
		if err != nil {
			log.Fatalf("ERROR: %v", err)
		}

		log.Printf("SN: %v", s)

	},
}
