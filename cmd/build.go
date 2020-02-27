/*
Copyright © 2020 Wayne Warren

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/waynr/image-generator/pkg/image"
)

// buildCmd represents the build command
var (
	layerSize  uint
	layerCount uint
	seed       = int64(4848484)
	tags       = []string{
		"registry.digitalocean.com/meow/rando",
	}
	buildCmd = &cobra.Command{
		Use:   "build",
		Short: "A brief description of your command",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			randomImageFactory := image.RandomImageFactory{
				LayerSizeKB: layerSize,
				LayerCount:  layerCount,
				Seed:        seed,
				Tags:        []string{"registry.digitalocean.com/meow/rando"},
			}

			return randomImageFactory.GenerateImage()
		},
	}
)

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().Int64VarP(&seed, "seed", "s", seed, "seed used to generate random layer contents")
	buildCmd.Flags().StringSliceVarP(&tags, "tags", "t", tags, "tags for generated image")

	buildCmd.Flags().UintVarP(&layerCount, "layer-count", "", layerCount, "image layer count")
	buildCmd.MarkFlagRequired("layer-count")
	buildCmd.Flags().UintVarP(&layerSize, "layer-size", "", layerSize, "image layer size in KB")
	buildCmd.MarkFlagRequired("layer-size")
}
