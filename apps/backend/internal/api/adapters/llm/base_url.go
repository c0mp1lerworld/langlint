package llm

// DefaultBaseURL is the OpenAI API endpoint used when OPENAI_BASE_URL is not
// set. The official SDK reads OPENAI_BASE_URL from the environment and, because
// it distinguishes "present but empty" from "unset" (os.LookupEnv), an empty
// value in .env would override the SDK default and produce relative requests
// ("unsupported protocol scheme"). Always passing an explicit base URL keeps the
// client usable when the override is blank.
const DefaultBaseURL = "https://api.openai.com/v1"

// BaseURLOrDefault returns baseURL, or DefaultBaseURL when baseURL is empty.
func BaseURLOrDefault(baseURL string) string {
	if baseURL == "" {
		return DefaultBaseURL
	}
	return baseURL
}
