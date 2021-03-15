package cmd

import (
	"fmt"
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
		d, err := c.RetrieveDesc()
		if err != nil {
			log.Fatalf("ERROR: %v", err)
		}

		// Reads goldleaf's version number
		s, err := c.RetrieveSerialNumber()
		if err != nil {
			log.Fatalf("ERROR: %v", err)
		}

		fmt.Printf(
			`
###################################
######## < < Q U A R K > > ########
###################################

goQuark is ready for connections...

+-----------------------------------+
|	Client:		%v    |
|	Version:	%v       |
+-----------------------------------+
`, d, s)

		for {
			// Magic [:4]
			i, err := c.ReadInt32()
			if err != nil {
				log.Fatalf("ERROR: %v", err)
			}

			if i != usbUtils.GLCI {
				log.Fatalf("ERROR: Invalid magic GLCI, got %v", i)
			}

			// CMD [4:]
			cmd, err := c.ReadCMD()
			if err != nil {
				log.Fatalln(err)
			}

			switch cmd {
			case usbUtils.Invalid:
				log.Printf("usbUtils.Invalid:")
			case usbUtils.GetDriveCount:
				log.Printf("usbUtils.SendDriveCount:")
				c.SendDriveCount()
			case usbUtils.GetDriveInfo:
				log.Printf("usbUtils.SendDriveInfo:")
				c.SendDriveInfo()
			case usbUtils.StatPath:
				log.Printf("usbUtils.StatPath:")
			case usbUtils.GetFileCount:
				log.Printf("usbUtils.GetFileCount:")
			case usbUtils.GetFile:
				log.Printf("usbUtils.GetFile:")
			case usbUtils.GetDirectoryCount:
				log.Printf("usbUtils.GetDirectoryCount:")
				c.SendDirectoryCount()
			case usbUtils.GetDirectory:
				log.Printf("usbUtils.GetDirectory:")
			case usbUtils.StartFile:
				log.Printf("usbUtils.StartFile:")
			case usbUtils.ReadFile:
				log.Printf("usbUtils.ReadFile:")
			case usbUtils.WriteFile:
				log.Printf("usbUtils.WriteFile:")
			case usbUtils.EndFile:
				log.Printf("usbUtils.EndFile:")
			case usbUtils.Create:
				log.Printf("usbUtils.Create:")
			case usbUtils.Delete:
				log.Printf("usbUtils.Delete:")
			case usbUtils.Rename:
				log.Printf("usbUtils.Rename:")
			case usbUtils.GetSpecialPathCount:
				log.Printf("usbUtils.SendSpecialPathCount:")
				c.SendSpecialPathCount()
			case usbUtils.GetSpecialPath:
				log.Printf("usbUtils.SendSpecialPath:")
			case usbUtils.SelectFile:
				log.Printf("usbUtils.SendSelectFile:")
				c.SendSelectFile()
			default:
				log.Printf("usbUtils.default:")
			}
		}

	},
}
