package update

import (
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	"github.com/EliasLd/goweeb/internal/version"
)

// Offer runs the startup update check and, when the user accepts the
// offer, installs the release. It never holds the program back: a
// failing or declined flow simply continues the normal startup.
//
// Offer is for interactive surfaces (reading the prompt off the
// shared stdin); the caller decides whether the check is enabled at
// all (flag, develop builds).
func Offer(
	currentVersion string,
	flavor Flavor,
	log *logger.Logger,
) error {
	if version.IsDevelop() {
		if log != nil {
			log.Debug(
				"No update check: %s is not a release build\n",
				currentVersion,
			)
		}

		return nil
	}

	release, err := Check(currentVersion, flavor, log)
	if err != nil {
		if log != nil {
			log.Debug("No update check: %v\n", err)
		}

		return nil
	}

	if release == nil {
		return nil
	}

	if log != nil {
		log.Info(
			"A new version is available: %s (you are on %s).\n",
			release.Tag,
			currentVersion,
		)
	}

	fmt.Print("Update now? [y/N] ")

	answer, _ := common.StdinReader.ReadString('\n')

	if !wantsUpdate(answer) {
		if log != nil {
			log.Info(
				"Staying on %s.\n",
				currentVersion,
			)
		}

		return nil
	}

	if err := Apply(release, log); err != nil {
		if log != nil {
			log.Error("The update failed: %v\n", err)
		}

		return err
	}

	return nil
}

// wantsUpdate reads the y/N answer; anything unconfirmed keeps the
// running version, including a hung-up prompt.
func wantsUpdate(answer string) bool {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true

	default:
		return false
	}
}
