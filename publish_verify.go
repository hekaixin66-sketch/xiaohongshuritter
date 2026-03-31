package main

import (
	"context"
	"fmt"
	"strings"
)

func (s *AppServer) VerifyPublishedNote(ctx context.Context, req *VerifyPublishedNoteRequest) (*VerifyPublishedNoteResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("verify request is required")
	}

	response := &VerifyPublishedNoteResponse{
		JobID:     strings.TrimSpace(req.JobID),
		NoteID:    strings.TrimSpace(req.NoteID),
		FeedID:    strings.TrimSpace(req.FeedID),
		XsecToken: strings.TrimSpace(req.XsecToken),
	}

	if response.JobID != "" {
		jobStatus, err := s.jobManager.Get(response.JobID)
		if err != nil {
			return nil, err
		}
		if jobStatus.Result != nil {
			if jobStatus.Result.TenantID != "" || jobStatus.Result.AccountID != "" {
				ctx = WithAccountScope(ctx, AccountScope{
					TenantID:  jobStatus.Result.TenantID,
					AccountID: jobStatus.Result.AccountID,
				})
			}
			response.NoteID = firstNonEmpty(response.NoteID, jobStatus.Result.NoteID)
			response.NoteURL = firstNonEmpty(response.NoteURL, jobStatus.Result.NoteURL)
			response.FeedID = firstNonEmpty(response.FeedID, jobStatus.Result.FeedID)
			response.XsecToken = firstNonEmpty(response.XsecToken, jobStatus.Result.XsecToken)
			response.AccountID = firstNonEmpty(response.AccountID, jobStatus.Result.AccountID)
			response.PublishedAt = firstNonEmpty(response.PublishedAt, jobStatus.Result.PublishedAt)
			response.ProductBindingResult = jobStatus.Result.ProductBindingResult
			response.PublishVerificationResult = jobStatus.Result.PublishVerificationResult
		}
	}

	if response.FeedID == "" || response.XsecToken == "" {
		notes, err := s.xiaohongshuService.ListRecentPublishedNotes(ctx, &RecentPublishedNotesRequest{
			AccountScope: AccountScopeFromContext(ctx),
			Limit:        20,
		})
		if err == nil {
			for _, note := range notes.Notes {
				if response.NoteID != "" && (note.NoteID == response.NoteID || note.FeedID == response.NoteID) {
					response.NoteID = firstNonEmpty(response.NoteID, note.NoteID)
					response.NoteURL = firstNonEmpty(response.NoteURL, note.NoteURL)
					response.FeedID = firstNonEmpty(response.FeedID, note.FeedID)
					response.XsecToken = firstNonEmpty(response.XsecToken, note.XsecToken)
					response.PublishedAt = firstNonEmpty(response.PublishedAt, note.PublishTime)
					break
				}
			}
		}
	}

	if response.FeedID == "" || response.XsecToken == "" {
		response.VerifyStatus = "pending"
		response.VerifyReason = "missing_feed_target_for_verification"
		return response, nil
	}

	detail, err := s.xiaohongshuService.getFeedDetailInternal(ctx, response.FeedID, response.XsecToken)
	if err != nil {
		response.VerifyStatus = "failed"
		response.VerifyReason = err.Error()
		return response, nil
	}

	response.NoteID = firstNonEmpty(response.NoteID, detail.Note.NoteID, response.FeedID)
	response.NoteURL = firstNonEmpty(response.NoteURL, buildFeedNoteURL(response.FeedID, response.XsecToken))
	response.PublishedAt = firstNonEmpty(response.PublishedAt, unixToRFC3339(detail.Note.Time))
	response.PublishVerificationResult = evaluatePublishVerification(detail, response.ProductBindingResult)
	if response.VerifyStatus == "" {
		response.VerifyStatus = "verified"
	}
	return response, nil
}
