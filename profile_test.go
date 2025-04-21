package iccarus

import (
	"fmt"
	"github.com/go-andiamo/iccarus/_test_data/profiles"
	"github.com/stretchr/testify/require"
	"slices"
	"strings"
	"testing"
)

func TestParseProfile(t *testing.T) {
	names := profiles.List()
	tagTypes := map[string]bool{}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			f, err := profiles.Open(name)
			require.NoError(t, err)
			defer func() {
				_ = f.Close()
			}()
			p, err := ParseProfile(f, &ParseOptions{Mode: ParseFull, ErrorOnUnknownTags: true, ErrorOnTagDecode: true})
			require.NoError(t, err)
			require.NotNil(t, p)
			for _, e := range p.TagHeaderTable.Entries {
				tagTypes[e.Signature] = true
			}
		})
	}
	keys := []string{}
	for k := range tagTypes {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	fmt.Printf("%s\n", strings.Join(keys, "\n"))
	fmt.Printf("%d sigs\n", len(tagSigs))
	for k := range tagSigs {
		fmt.Printf("'%s'\n", k)
	}
}

func TestSharedTagOffsets(t *testing.T) {
	names := profiles.List()
	shared := 0
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			f, err := profiles.Open(name)
			require.NoError(t, err)
			defer func() {
				_ = f.Close()
			}()

			profile, err := ParseProfile(f, nil)
			require.NoError(t, err)

			offsets := map[uint32][]string{}
			for _, tag := range profile.TagHeaderTable.Entries {
				offsets[tag.Offset] = append(offsets[tag.Offset], tag.Signature)
			}

			for offset, sigs := range offsets {
				if len(sigs) > 1 {
					shared++
					t.Logf("Profile %s: tags %v share offset 0x%X", name, sigs, offset)
				}
			}
		})
	}
	fmt.Printf("Shared occurences: %d\n", shared)
}
