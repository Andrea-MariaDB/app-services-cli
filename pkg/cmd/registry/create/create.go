package create

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redhat-developer/app-services-cli/pkg/localize"

	"github.com/redhat-developer/app-services-cli/pkg/cmd/flag"
	flagutil "github.com/redhat-developer/app-services-cli/pkg/cmdutil/flags"
	"github.com/redhat-developer/app-services-cli/pkg/connection"
	"github.com/redhat-developer/app-services-cli/pkg/kafka"

	srsmgmtv1 "github.com/redhat-developer/app-services-sdk-go/srsmgmt/apiv1/client"

	"github.com/AlecAivazis/survey/v2"
	"github.com/redhat-developer/app-services-cli/pkg/dump"
	"github.com/redhat-developer/app-services-cli/pkg/iostreams"

	pkgKafka "github.com/redhat-developer/app-services-cli/pkg/kafka"
	"github.com/redhat-developer/app-services-cli/pkg/logging"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/redhat-developer/app-services-cli/internal/config"
	"github.com/redhat-developer/app-services-cli/pkg/cmd/factory"
	"github.com/redhat-developer/app-services-cli/pkg/cmdutil"
)

type Options struct {
	name string

	outputFormat string
	autoUse      bool

	interactive bool

	IO         *iostreams.IOStreams
	Config     config.IConfig
	Connection factory.ConnectionFunc
	Logger     func() (logging.Logger, error)
	localizer  localize.Localizer
}

// NewCreateCommand creates a new command for creating kafkas.
func NewCreateCommand(f *factory.Factory) *cobra.Command {
	opts := &Options{
		IO:         f.IOStreams,
		Config:     f.Config,
		Connection: f.Connection,
		Logger:     f.Logger,
		localizer:  f.Localizer,
	}

	cmd := &cobra.Command{
		Use:     "create",
		Short:   f.Localizer.MustLocalize("registry.cmd.create.shortDescription"),
		Long:    f.Localizer.MustLocalize("registry.cmd.create.longDescription"),
		Example: f.Localizer.MustLocalize("registry.cmd.create.example"),
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.name = args[0]

				if err := kafka.ValidateName(opts.name); err != nil {
					return err
				}
			}

			if !opts.IO.CanPrompt() && opts.name == "" {
				return errors.New(opts.localizer.MustLocalize("registry.cmd.create.error.name.requiredWhenNonInteractive"))
			} else if opts.name == "" {
				opts.interactive = true
			}

			validOutputFormats := flagutil.ValidOutputFormats
			if opts.outputFormat != "" && !flagutil.IsValidInput(opts.outputFormat, validOutputFormats...) {
				return flag.InvalidValueError("output", opts.outputFormat, validOutputFormats...)
			}

			return runCreate(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.outputFormat, "output", "o", "json", opts.localizer.MustLocalize("registry.cmd.create.flag.output.description"))
	cmd.Flags().BoolVar(&opts.autoUse, "use", true, opts.localizer.MustLocalize("registry.cmd.create.flag.use.description"))

	return cmd
}

// nolint:funlen
func runCreate(opts *Options) error {
	logger, err := opts.Logger()
	if err != nil {
		return err
	}

	cfg, err := opts.Config.Load()
	if err != nil {
		return err
	}

	connection, err := opts.Connection(connection.DefaultConfigSkipMasAuth)
	if err != nil {
		return err
	}

	api := connection.API()

	var payload *srsmgmtv1.RegistryCreate
	if opts.interactive {
		logger.Debug()

		payload, err = promptPayload(opts)
		if err != nil {
			return err
		}
	} else {
		payload = &srsmgmtv1.RegistryCreate{
			Name: &opts.name,
		}
	}

	logger.Info("Creating service registry " + opts.name)

	a := api.ServiceRegistryMgmt().CreateRegistry(context.Background())
	a = a.RegistryCreate(*payload)
	response, _, err := a.Execute()

	if err != nil {
		return err
	}

	logger.Info(opts.localizer.MustLocalize("registry.cmd.create.info.successMessage"))

	switch opts.outputFormat {
	case "json":
		data, _ := json.MarshalIndent(response, "", cmdutil.DefaultJSONIndent)
		_ = dump.JSON(opts.IO.Out, data)
	case "yaml", "yml":
		data, _ := yaml.Marshal(response)
		_ = dump.YAML(opts.IO.Out, data)
	}

	registryConfig := &config.ServiceRegistryConfig{
		InstanceID: response.GetId(),
		Name:       *response.Name,
	}

	if opts.autoUse {
		// TODO Generic use
		logger.Debug(opts.localizer.MustLocalize("kafka.create.debug.autoUseSetMessage"))
		cfg.Services.ServiceRegistry = registryConfig
		if err := opts.Config.Save(cfg); err != nil {
			return fmt.Errorf("%v: %w", opts.localizer.MustLocalize("kafka.common.error.couldNotUseKafka"), err)
		}
	} else {
		logger.Debug(opts.localizer.MustLocalize("kafka.create.debug.autoUseNotSetMessage"))
	}

	return nil
}

// Show a prompt to allow the user to interactively insert the data for their Kafka
func promptPayload(opts *Options) (payload *srsmgmtv1.RegistryCreate, err error) {
	if err != nil {
		return nil, err
	}

	// set type to store the answers from the prompt with defaults
	answers := struct {
		Name string
	}{}

	promptName := &survey.Input{
		Message: opts.localizer.MustLocalize("registry.cmd.create.input.name.message"),
		Help:    opts.localizer.MustLocalize("registry.cmd.create.input.name.help"),
	}

	err = survey.AskOne(promptName, &answers.Name, survey.WithValidator(pkgKafka.ValidateName))
	if err != nil {
		return nil, err
	}

	payload = &srsmgmtv1.RegistryCreate{
		Name: &answers.Name,
	}

	return payload, nil
}
