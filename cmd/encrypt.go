package cmd

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/spf13/cobra"
)

// ── encrypt ───────────────────────────────────────────────────────────────────

var (
	encryptOutput   string
	encryptUserPW   string
	encryptOwnerPW  string
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt <input.pdf>",
	Short: "Password-protect a PDF",
	Long: `Encrypts a PDF with a user password (required to open) and optional owner
password (required to edit/print). Without -o, encrypts in-place.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if encryptUserPW == "" && encryptOwnerPW == "" {
			return fmt.Errorf("provide at least --user-pw or --owner-pw")
		}
		return runEncrypt(args[0])
	},
}

func init() {
	encryptCmd.Flags().StringVarP(&encryptOutput, "output", "o", "", "Output file (default: in-place)")
	encryptCmd.Flags().StringVar(&encryptUserPW, "user-pw", "", "User password (required to open)")
	encryptCmd.Flags().StringVar(&encryptOwnerPW, "owner-pw", "", "Owner password (required to modify)")
	rootCmd.AddCommand(encryptCmd)
}

func runEncrypt(inFile string) error {
	printInfo(fmt.Sprintf("Encrypting %s…", inFile))

	conf := model.NewDefaultConfiguration()
	conf.UserPW = encryptUserPW
	conf.OwnerPW = encryptOwnerPW
	if conf.OwnerPW == "" {
		conf.OwnerPW = conf.UserPW // sensible default
	}

	outFile := encryptOutput
	if outFile == "" {
		outFile = inFile
	}

	if err := api.EncryptFile(inFile, outFile, conf); err != nil {
		return err
	}

	fi, _ := os.Stat(outFile)
	size := ""
	if fi != nil {
		size = fmt.Sprintf(" (%s)", humanSize(fi.Size()))
	}
	printSuccess(fmt.Sprintf("Encrypted: %s%s", outFile, size))
	return nil
}

// ── decrypt ───────────────────────────────────────────────────────────────────

var (
	decryptOutput string
	decryptPW     string
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt <input.pdf>",
	Short: "Remove password protection from a PDF",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if decryptPW == "" {
			return fmt.Errorf("provide the password with --password")
		}
		return runDecrypt(args[0])
	},
}

func init() {
	decryptCmd.Flags().StringVarP(&decryptOutput, "output", "o", "", "Output file (default: in-place)")
	decryptCmd.Flags().StringVar(&decryptPW, "password", "", "PDF password")
	_ = decryptCmd.MarkFlagRequired("password")
	rootCmd.AddCommand(decryptCmd)
}

func runDecrypt(inFile string) error {
	printInfo(fmt.Sprintf("Decrypting %s…", inFile))

	conf := model.NewDefaultConfiguration()
	conf.UserPW = decryptPW
	conf.OwnerPW = decryptPW

	outFile := decryptOutput
	if outFile == "" {
		outFile = inFile
	}

	if err := api.DecryptFile(inFile, outFile, conf); err != nil {
		return err
	}

	fi, _ := os.Stat(outFile)
	size := ""
	if fi != nil {
		size = fmt.Sprintf(" (%s)", humanSize(fi.Size()))
	}
	printSuccess(fmt.Sprintf("Decrypted: %s%s", outFile, size))
	return nil
}
