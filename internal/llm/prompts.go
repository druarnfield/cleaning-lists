package llm

import (
	"fmt"
	"os"
	"strconv"
)

// GetSystemPrompt returns the system prompt for the LLM scheduler
func GetSystemPrompt() string {
	balanceThreshold := os.Getenv("BALANCE_THRESHOLD_MINS")
	if balanceThreshold == "" {
		balanceThreshold = "30"
	}

	return fmt.Sprintf(`You are a household task scheduler. Schedule tasks fairly between Dru and Josie.

RULES:
1. If default_assignee="dru" → assign to "dru" only
2. If default_assignee="josie" → assign to "josie" only
3. If default_assignee="both" → create TWO instances (one for "dru", one for "josie")
4. If default_assignee="alternate" or NULL → assign to either "dru" or "josie" (your choice)
5. Balance total weekly minutes between them (within ±%s minutes)
6. assigned_to must be exactly "dru" or "josie" (lowercase)

OUTPUT FORMAT (JSON only, no other text):
{
  "instances": [
    {
      "task_id": 1,
      "scheduled_date": "2024-12-02",
      "assigned_to": "dru",
      "week_start_date": "2024-12-02",
      "reasoning": "Dru has fewer minutes this week"
    }
  ],
  "summary": "Balanced workload: Dru 120min, Josie 115min"
}`, balanceThreshold)
}

// GetSchedulingPrompt creates the prompt for generating a new schedule
func GetSchedulingPrompt(context SchedulingContext, weeksAhead int) string {
	return fmt.Sprintf(`TASK: Create %d weeks of task instances from the data below.

%s

INSTRUCTIONS:
1. Look at "Next due" dates - if a task is due within the period, schedule it
2. For frequent tasks (occur multiple times in period), create appropriate instances
3. Follow the default_assignee rules from system prompt
4. Balance total minutes between Dru and Josie
5. Output JSON only (no extra text)

Examples:
- Task "Weekly (7 days)" → create 1 per week (4 instances for 4 weeks)
- Task "12 Weekly (84 days) | Next due: 2024-12-05 (5 days)" → create 1 instance on/near Dec 5
- Task "12 Weekly (84 days) | Next due: 2025-01-20 (51 days)" → skip (not due in this period)
- Task "2 x/week" → create 2 per week (8 instances for 4 weeks)

KEY: Use "Next due" date to determine if long-frequency tasks should be scheduled.

Response format: JSON object with "instances" array and "summary" string.`, weeksAhead, context.Format(), weeksAhead)
}

// GetAdjustmentPrompt creates the prompt for adjusting an existing schedule
func GetAdjustmentPrompt(context SchedulingContext, userMessage string, weekOffset int, weeksAhead int) string {
	weekLabel := "this week"
	if weekOffset == 1 {
		weekLabel = "next week"
	} else if weekOffset > 1 {
		weekLabel = fmt.Sprintf("%d weeks from now", weekOffset)
	} else if weekOffset == -1 {
		weekLabel = "last week"
	} else if weekOffset < -1 {
		weekLabel = fmt.Sprintf("%d weeks ago", -weekOffset)
	}

	return fmt.Sprintf(`TASK: Adjust the schedule based on user request.

%s

USER REQUEST (viewing %s): %s

WHAT YOU CAN DO:
- Change who is assigned to tasks (dru ↔ josie)
- Move tasks to different dates
- Add or remove task instances
- Adjust across all %d weeks shown above

IMPORTANT:
- Instances shown = non-completed only (completed tasks already filtered out)
- Still respect default_assignee rules unless user explicitly overrides
- User's request takes priority over balance
- Output JSON only (no extra text)

RESPONSE FORMAT:
{
  "instances": [array of all instances for the %d weeks],
  "summary": "explanation of what you changed"
}

Optional - only if user wants to create/modify/delete task definitions:
{
  "instances": [...],
  "task_changes": [
    {"action": "create", "task_id": null, "name": "New Task", "category": "Kitchen", "frequency": "Weekly", "estimated_mins": 15, "default_assignee": "dru"}
  ],
  "summary": "..."
}`, context.Format(), weekLabel, userMessage, weeksAhead, weeksAhead)
}

// ParseWeeksAhead gets the weeks ahead configuration from environment
func ParseWeeksAhead() int {
	weeksAheadStr := os.Getenv("WEEKS_AHEAD")
	if weeksAheadStr == "" {
		return 4 // Default
	}

	weeksAhead, err := strconv.Atoi(weeksAheadStr)
	if err != nil || weeksAhead < 1 {
		return 4 // Default on error
	}

	return weeksAhead
}
