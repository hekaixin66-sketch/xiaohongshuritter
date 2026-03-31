package main

import (
	"testing"

	"github.com/hekaixin66-sketch/xiaohongshuritter/xiaohongshu"
)

func TestEvaluatePublishVerificationNoProducts(t *testing.T) {
	result := evaluatePublishVerification(&xiaohongshu.FeedDetailResponse{
		Note: xiaohongshu.FeedDetail{
			Type:      "normal",
			ImageList: []xiaohongshu.DetailImageInfo{{URLDefault: "a"}},
		},
	}, ProductBindingResult{})

	if !result.PublishVisible {
		t.Fatal("expected publish visible")
	}
	if !result.CoverVisible {
		t.Fatal("expected cover visible")
	}
	if result.VerifyStatus != "verified" {
		t.Fatalf("unexpected verify status: %s", result.VerifyStatus)
	}
}

func TestEvaluatePublishVerificationProductFailure(t *testing.T) {
	result := evaluatePublishVerification(&xiaohongshu.FeedDetailResponse{
		Note: xiaohongshu.FeedDetail{
			Type:      "normal",
			ImageList: []xiaohongshu.DetailImageInfo{{URLDefault: "a"}},
		},
	}, ProductBindingResult{
		Status:            "failed",
		ProductsRequested: []string{"空调"},
		ProductsMissing:   []string{"空调"},
	})

	if result.ProductVisible == nil {
		t.Fatal("expected product visibility decision")
	}
	if *result.ProductVisible {
		t.Fatal("expected product visible to be false")
	}
	if result.VerifyStatus != "partial" {
		t.Fatalf("unexpected verify status: %s", result.VerifyStatus)
	}
}
