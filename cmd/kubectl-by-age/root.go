package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/alecjacobs5401/kubectl-by-age/pkg/timeago"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes/scheme"

	_ "k8s.io/client-go/plugin/pkg/client/auth" // combined authprovider import
)

var (
	cf      *genericclioptions.ConfigFlags
	rbf     *genericclioptions.ResourceBuilderFlags
	pf      *genericclioptions.PrintFlags
	rootCmd = &cobra.Command{
		Use:          "kubectl by-age <resource> [<flags>]",
		SilenceUsage: true, // for when RunE returns an error
		Short:        "Filter and sort Kubernetes resources by their age",
		Example: `  kubectl by-age pod
  kubectl by-age pod,deploy,service
  kubectl by-age deploy -m 50d
  kubectl by-age service -m 20d4h -M 21d4h5m`,
		RunE:    run,
		Version: version,
	}
	minAge      string
	maxAge      string
	reverseSort bool
	resources   []string
)

const version = "" // auto-populated by goreleaser
const durationExplanation = `Represented as a string duration of age, where the duration is subtracted from the current time.
For example, '1d20h5m' represents 1 day, 20 hours, and 5 minutes ago.
Accepts time spans of year (y), month (M), day (d), hour (h), minute (m), and second (s)`

func run(command *cobra.Command, resources []string) error {
	printer, err := printer()
	if err != nil {
		return err
	}
	var writer io.Writer = os.Stdout
	if _, ok := printer.(*printers.HumanReadablePrinter); ok {
		w := printers.GetNewTabWriter(writer)
		writer = w
		defer w.Flush()
	}

	var maxAgeTime time.Time // "nil" Time value
	minAgeTime := time.Now()

	if minAge != "" {
		minAgeTime, err = timeago.Parse(minAge)
		if err != nil {
			return errors.Wrap(err, "Parsing min-age")
		}
	}
	if maxAge != "" {
		maxAgeTime, err = timeago.Parse(maxAge)
		if err != nil {
			return errors.Wrap(err, "Parsing max-age")
		}
	}

	err = rbf.ToBuilder(cf, resources).Do().Visit(func(info *resource.Info, err error) error {
		obj := info.Object

		acc, _ := meta.Accessor(obj)
		if err != nil {
			return errors.Wrapf(err, "could not determine object type for", info.ObjectName())
		}

		createdAt := acc.GetCreationTimestamp().Time
		if createdAt.Before(minAgeTime) && createdAt.After(maxAgeTime) {
			printer.PrintObj(info.Object, writer)
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

func init() {
	rootCmd.Flags().StringVarP(&minAge, "min-age", "m", "", fmt.Sprintf("Define the minimum age of resource to return.%s", durationExplanation))
	rootCmd.Flags().StringVarP(&maxAge, "max-age", "M", "", fmt.Sprintf("Define the maximum age of resource to return.%s", durationExplanation))
	rootCmd.Flags().BoolVarP(&reverseSort, "reverse-sort", "r", false, "Sort by ascending age. Defaults to sorting by descending age.")

	cf = genericclioptions.NewConfigFlags(true)
	cf.AddFlags(rootCmd.Flags())
	rbf = genericclioptions.NewResourceBuilderFlags()
	rbf.WithLabelSelector("")
	rbf.WithFieldSelector("")
	rbf.WithAllNamespaces(false)
	rbf.WithAll(true)
	rbf.AddFlags(rootCmd.Flags())
	pf = genericclioptions.NewPrintFlags("")
	pf.WithTypeSetter(scheme.Scheme).WithDefaultOutput("status")
	pf.AddFlags(rootCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func printer() (printers.ResourcePrinter, error) {
	options := printers.PrintOptions{
		WithNamespace: *rbf.AllNamespaces,
		WithKind:      true,
		Wide:          *pf.OutputFormat == "wide",
	}
	if pf.OutputFlagSpecified() && !options.Wide {
		return pf.ToPrinter()
	}
	return printers.NewTablePrinter(options), nil
}
