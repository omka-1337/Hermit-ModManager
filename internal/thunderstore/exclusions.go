package thunderstore

import (
	"context"
	_ "embed"
	"errors"
	"strings"
	"time"
)

// ExclusionsURL is r2modman's list of packages that are not installable mods
// (mod managers and similar tools), shared by Thunderstore mod managers.
const ExclusionsURL = "https://raw.githubusercontent.com/ebkr/r2modmanPlus/master/modExclusions.md"

// defaultExclusions is a bundled copy of ExclusionsURL (r2modman, MIT
// license), used until the online list has been fetched.
//
//go:embed modExclusions.md
var defaultExclusions string

func parseExclusions(list string) map[string]bool {
	set := map[string]bool{}
	for _, line := range strings.Split(list, "\n") {
		if name := strings.TrimSpace(line); name != "" {
			set[name] = true
		}
	}
	return set
}

// Exclusions returns full package names ("<namespace>-<name>") hidden from
// listings. It never fails: the bundled list is used when offline.
func (c *Client) Exclusions(ctx context.Context) map[string]bool {
	c.exclusionsMu.Lock()
	defer c.exclusionsMu.Unlock()

	if c.exclusions != nil && time.Since(c.exclusionsFetched) < schemaTTL {
		return c.exclusions
	}
	data, err := c.cachedFetch(ctx, "mod-exclusions.md", c.exclusionsURL, func(b []byte) error {
		if len(parseExclusions(string(b))) == 0 {
			return errors.New("empty exclusion list")
		}
		return nil
	})
	if err != nil {
		c.exclusions = parseExclusions(defaultExclusions)
	} else {
		c.exclusions = parseExclusions(string(data))
	}
	c.exclusionsFetched = time.Now()
	return c.exclusions
}
