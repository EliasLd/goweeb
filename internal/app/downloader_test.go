package app

import (
	"bytes"
	"testing"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// stubProvider is the minimal provider contract, with no optional
// capability attached.
type stubProvider struct{}

func (p *stubProvider) Name() string { return "stub" }

func (p *stubProvider) Search(
	_ string,
	_ *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	return nil, nil
}

func (p *stubProvider) ListScanPaths(
	_ string,
	_ *logger.Logger,
) ([]common.SelectableItem, error) {
	return nil, nil
}

func (p *stubProvider) ListEntries(
	_ string,
	_ string,
	_ *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	return sourcetypes.Work{}, nil, nil
}

func (p *stubProvider) GetPageImageURLs(
	_ string,
	_ *logger.Logger,
) ([]string, error) {
	return nil, nil
}

// stubDecoder adds the optional decode capability on top.
type stubDecoder struct {
	stubProvider
}

func (d *stubDecoder) DecodeImageData(
	_ string,
	data []byte,
) ([]byte, bool) {
	return bytes.ToUpper(data), true
}

func TestPageTransformResolvesOptionalDecoder(t *testing.T) {
	if got := pageTransform(&stubProvider{}); got != nil {
		t.Errorf(
			"expected no transform without the capability, got %v",
			got,
		)
	}

	transform := pageTransform(&stubDecoder{})
	if transform == nil {
		t.Fatal("expected the decoder to adapt into a transform")
	}

	decoded, ok := transform("https://site/page", []byte("cipher"))
	if !ok {
		t.Fatal("expected the transform to claim the page")
	}

	if !bytes.Equal(decoded, []byte("CIPHER")) {
		t.Errorf(
			"transform() = %q, want the decoded bytes",
			decoded,
		)
	}
}
