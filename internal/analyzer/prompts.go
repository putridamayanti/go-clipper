package analyzer

import (
	"fmt"
	"go-clipper/internal/dtos"
)

// analyzePrompt builds the segment-finding prompt shared by every provider.
// intro describes what the model is given (the video itself or a transcript).
func analyzePrompt(payload dtos.AnalyzeRequest, intro string) string {
	startTs := payload.StartTimestamp
	if startTs == "" {
		startTs = "start"
	}
	endTs := payload.EndTimestamp
	if endTs == "" {
		endTs = "end"
	}

	return fmt.Sprintf(`
		%s
		1. Identify exactly %d of the most engaging and "viral" segments that would make great short-form clips (TikTok, Reels, Shorts).
		2. Ensure each segment is at least %d seconds long and maximum %d seconds.
		3. For each segment, provide the start and end timestamps (in seconds). THESE ARE MANDATORY.
		4. Provide a brief description and a compelling "hook" title for each clip. THE HOOK MUST NOT BE EMPTY.
		5. Analyze only from duration %s to %s

		Output the result as a single JSON object in the following format:
		{
			"description": "Short summary of the video",
			"segments": [
				{
					"start": "30",
					"end": "60",
					"description": "Deep insight about AI",
					"hook": "The Truth About AI"
				}
			]
		}
	`, intro, payload.Count, payload.MinimumDuration, payload.MaximumDuration, startTs, endTs)
}

// descriptionPrompt builds the caption/description prompt shared by every provider.
func descriptionPrompt(intro string) string {
	return fmt.Sprintf(`
		%s
		1. Include the tags to separate by commas without # with At least 400 characters. MANDATORY. Make sure at leas 400 characters. Make sure it will help content viral.
		2. No timestamp/seconds in the description.
		3. Will repost this in youtube short and instagram

		Output the result as a single JSON object in the following format:
		{
			"description": "The chaos continues in this TRIP",
			"tags": "viral video, funny, fun"
		}
	`, intro)
}
