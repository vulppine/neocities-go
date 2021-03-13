package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vulppine/neocities-go"
)

func AddNeoCitiesCMD(c *cobra.Command) {
	c.AddCommand(NeoCitiesCMD)
}

func init() {
	NeoCitiesCMD.Flags().String("key", "", "The API key of your NeoCities site.")
	NeoCitiesCMD.Flags().String("keyfile", "", "The API keyfile of your NeoCities site.")
	NeoCitiesCMD.AddCommand(UploadCMD)
	NeoCitiesCMD.AddCommand(PushCMD)
	NeoCitiesCMD.AddCommand(DeleteCMD)

	UploadCMD.Flags().String("name", "", "The name you want for your uploaded file.")
}

var (
	err error
	s = neocities.Site{}

	k string
	n string
	NeoCitiesCMD = &cobra.Command{
		Use: "neocities",
	}

	UploadCMD = &cobra.Command{
		Use: "upload file [-name string]",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if k != "" {
				s.Key, err = neocities.ReadFile(k)
				if err != nil {
					return fmt.Errorf("keyfile read returned error")
				}
			}

			if n == "" {
				n = filepath.Base(args[0])
			}

			err := s.UploadFile(args[0], n, nil)
			if err != nil {
				return err
			}

			return nil
		},
	}

	PushCMD = &cobra.Command{
		Use: "push directory",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if k != "" {
				s.Key, err = neocities.ReadFile(k)
				if err != nil {
					return fmt.Errorf("keyfile read returned error")
				}
			}

			err := s.Push(args[0], nil)
			if err != nil {
				return err
			}

			return nil
		},
	}

	DeleteCMD = &cobra.Command{
		Use: "delete files",
		Args: cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if k != "" {
				s.Key, err = neocities.ReadFile(k)
				if err != nil {
					return fmt.Errorf("keyfile read returned error")
				}
			}

			err := s.DeleteFiles(nil, args...)
			if err != nil {
				return nil
			}

			return nil
		},
	}
)
