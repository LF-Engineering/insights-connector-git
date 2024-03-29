package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LF-Engineering/insights-datasource-git/build"
	shared "github.com/LF-Engineering/insights-datasource-shared"
	"github.com/LF-Engineering/insights-datasource-shared/auth0"
	"github.com/LF-Engineering/insights-datasource-shared/aws"
	"github.com/LF-Engineering/insights-datasource-shared/cache"
	elastic "github.com/LF-Engineering/insights-datasource-shared/elastic"
	"github.com/LF-Engineering/insights-datasource-shared/http"
	logger "github.com/LF-Engineering/insights-datasource-shared/ingestjob"
	"github.com/LF-Engineering/insights-datasource-shared/report"
	"github.com/LF-Engineering/lfx-event-schema/service"
	"github.com/LF-Engineering/lfx-event-schema/service/insights"
	"github.com/LF-Engineering/lfx-event-schema/service/insights/git"
	"github.com/LF-Engineering/lfx-event-schema/service/repository"
	"github.com/LF-Engineering/lfx-event-schema/service/user"
	"github.com/LF-Engineering/lfx-event-schema/utils/datalake"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-git/go-billy/v5/osfs"
	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gitCache "github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

const (
	// GitBackendVersion - backend version
	GitBackendVersion = "0.1.1"
	// GitDefaultReposPath - default path where git repository clones
	GitDefaultReposPath = "/tmp/git-repositories"
	// GitDefaultCachePath - default path where gitops cache files are stored
	GitDefaultCachePath = "/tmp/git-cache"
	// GitOpsCommand - command that maintains git stats cache
	// GitOpsCommand = "gitops.py"
	GitOpsCommand = "gitops"
	// GitOpsFailureFatal - is GitOpsCommand failure fatal?
	GitOpsFailureFatal = true
	// OrphanedCommitsCommand - command to list orphaned commits
	OrphanedCommitsCommand = "detect-removed-commits.sh"
	// OrphanedCommitsFailureFatal - is OrphanedCommitsCommand failure fatal?
	OrphanedCommitsFailureFatal = true
	// GitParseStateInit - init parser state
	GitParseStateInit = 0
	// GitParseStateCommit - commit parser state
	GitParseStateCommit = 1
	// GitParseStateHeader - header parser state
	GitParseStateHeader = 2
	// GitParseStateMessage - message parser state
	GitParseStateMessage = 3
	// GitParseStateFile - file parser state
	GitParseStateFile = 4
	// GitCommitDateField - date field in the commit structure
	GitCommitDateField = "CommitDate"
	// GitDefaultSearchField - default search field
	GitDefaultSearchField = "item_id"
	// GitHubURL - GitHub URL
	GitHubURL = "https://github.com/"
	// GitMaxCommitProperties - maximum properties that can be set on the commit object
	GitMaxCommitProperties = 1000
	// GitMaxMsgLength - maximum message length
	GitMaxMsgLength = 0x4000
	// GitGenerateFlatDocs - do we want to generate flat commit co-authors docs, like docs with type: commit_co_author, commit_signer etc.?
	GitGenerateFlatDocs = true
	// GitDefaultStream - Stream To Publish
	GitDefaultStream = "PUT-S3-git-commits"
	// GitDataSource - constant for git source
	GitDataSource = "git"
	// UnknownExtension - Empty file extension type
	UnknownExtension = "UNKNOWN"
	// CommitCreated commit created event
	CommitCreated = "commit.created"
	// CommitUpdated commit updated event
	CommitUpdated = "commit.updated"
	// InProgress status
	InProgress = "in_progress"
	// Failed status
	Failed = "failed"
	// Success status
	Success = "success"
	// GitConnector ...
	GitConnector   = "git-connector"
	PackSize       = 1000
	HotRepoCount   = 50000
	YearFirstHalf  = "first-half"
	YearSecondHalf = "second-half"
)

var (
	// GitCategories - categories defined for git
	GitCategories = map[string]struct{}{"commit": {}}
	// GitDefaultEnv - default git command environment
	GitDefaultEnv = map[string]string{"LANG": "C", "PAGER": ""}
	// GitLogOptions - default git log options
	GitLogOptions = []string{
		"--raw",           // show data in raw format
		"--numstat",       // show added/deleted lines per file
		"--pretty=fuller", // pretty output
		"--decorate=full", // show full refs
		"--parents",       //show parents information
		"-M",              //detect and report renames
		"-C",              //detect and report copies
		"-c",              //show merge info
	}
	// GitCommitPattern - pattern to match a commit
	GitCommitPattern = regexp.MustCompile(`^commit[ \t](?P<commit>[a-f0-9]{40})(?:[ \t](?P<parents>[a-f0-9][a-f0-9 \t]+))?(?:[ \t]\((?P<refs>.+)\))?$`)
	// GitHeaderPattern - pattern to match a commit
	GitHeaderPattern = regexp.MustCompile(`^(?P<name>[a-zA-z0-9\-]+)\:[ \t]+(?P<value>.+)$`)
	// GitMessagePattern - message patterns
	GitMessagePattern = regexp.MustCompile(`^[\s]{4}(?P<msg>.*)$`)
	// GitTrailerPattern - message trailer pattern
	GitTrailerPattern = regexp.MustCompile(`^(?P<name>[a-zA-z0-9\-]+)\:[ \t]+(?P<value>.+)$`)
	// GitActionPattern - action pattern - note that original used `\.{,3}` which is not supported in go - you must specify from=0: `\.{0,3}`
	GitActionPattern = regexp.MustCompile(`^(?P<sc>\:+)(?P<modes>(?:\d{6}[ \t])+)(?P<indexes>(?:[a-f0-9]+\.{0,3}[ \t])+)(?P<action>[^\t]+)\t+(?P<file>[^\t]+)(?:\t+(?P<newfile>.+))?$`)
	// GitStatsPattern - stats pattern
	GitStatsPattern = regexp.MustCompile(`^(?P<added>\d+|-)[ \t]+(?P<removed>\d+|-)[ \t]+(?P<file>.+)$`)
	// GitAuthorsPattern - author pattern
	// Example: David Woodhouse <dwmw2@infradead.org> and Tilman Schmidt <tilman@imap.cc>
	GitAuthorsPattern = regexp.MustCompile(`(?P<first_authors>.* .*) and (?P<last_author>.* .*) (?P<email>.*)`)
	// GitCoAuthorsPattern - author pattern
	// Example: Co-authored-by: Andi Kleen <ak@suse.de>
	GitCoAuthorsPattern = regexp.MustCompile(`Co-authored-by:(?P<first_authors>.* .*)<(?P<email>.*)>\n?`)
	// GitDocFilePattern - files matching this pattern are detected as documentation files, so commit will be marked as doc_commit
	GitDocFilePattern = regexp.MustCompile(`(?i)(\.md$|\.rst$|\.docx?$|\.txt$|\.pdf$|\.jpe?g$|\.png$|\.svg$|\.img$|^docs/|^documentation/|^readme)`)
	// GitCommitRoles - roles to fetch affiliation data
	GitCommitRoles = []string{"Author", "Commit"}
	// GitAllowedTrailers - allowed commit trailer flags (lowercase/case insensitive -> correct case)
	GitAllowedTrailers = map[string][]string{
		"about-fscking-timed-by":                 {"Reviewed-by"},
		"accked-by":                              {"Reviewed-by"},
		"aced-by":                                {"Reviewed-by"},
		"ack":                                    {"Reviewed-by"},
		"ack-by":                                 {"Reviewed-by"},
		"ackde-by":                               {"Reviewed-by"},
		"acked":                                  {"Reviewed-by"},
		"acked-and-reviewed":                     {"Reviewed-by"},
		"acked-and-reviewed-by":                  {"Reviewed-by"},
		"acked-and-tested-by":                    {"Reviewed-by", "Tested-by"},
		"acked-b":                                {"Reviewed-by"},
		"acked-by":                               {"Reviewed-by"},
		"acked-by-stale-maintainer":              {"Reviewed-by"},
		"acked-by-with-comments":                 {"Reviewed-by"},
		"acked-by-without-testing":               {"Reviewed-by"},
		"acked-for-mfd-by":                       {"Reviewed-by"},
		"acked-for-now-by":                       {"Reviewed-by"},
		"acked-off-by":                           {"Reviewed-by"},
		"acked-the-net-bits-by":                  {"Reviewed-by"},
		"acked-the-tulip-bit-by":                 {"Reviewed-by"},
		"acked-with-apologies-by":                {"Reviewed-by"},
		"acked_by":                               {"Reviewed-by"},
		"ackedby":                                {"Reviewed-by"},
		"ackeded-by":                             {"Reviewed-by"},
		"acknowledged-by":                        {"Reviewed-by"},
		"acted-by":                               {"Reviewed-by"},
		"actually-written-by":                    {"Co-authored-by"},
		"additional-author":                      {"Co-authored-by"},
		"all-the-fault-of":                       {"Informed-by"},
		"also-analyzed-by":                       {"Reviewed-by"},
		"also-fixed-by":                          {"Co-authored-by"},
		"also-posted-by":                         {"Reported-by"},
		"also-reported-and-tested-by":            {"Reported-by", "Tested-by"},
		"also-reported-by":                       {"Reported-by"},
		"also-spotted-by":                        {"Reported-by"},
		"also-suggested-by":                      {"Reviewed-by"},
		"also-written-by":                        {"Co-authored-by"},
		"analysed-by":                            {"Reviewed-by"},
		"analyzed-by":                            {"Reviewed-by"},
		"aoled-by":                               {"Reviewed-by"},
		"apology-from":                           {"Informed-by"},
		"appreciated-by":                         {"Informed-by"},
		"approved":                               {"Approved-by"},
		"approved-by":                            {"Approved-by"},
		"architected-by":                         {"Influenced-by"},
		"assisted-by":                            {"Co-authored-by"},
		"badly-reviewed-by ":                     {"Reviewed-by"},
		"based-in-part-on-patch-by":              {"Influenced-by"},
		"based-on":                               {"Influenced-by"},
		"based-on-a-patch-by":                    {"Influenced-by"},
		"based-on-code-by":                       {"Influenced-by"},
		"based-on-code-from":                     {"Influenced-by"},
		"based-on-comments-by":                   {"Influenced-by"},
		"based-on-idea-by":                       {"Influenced-by"},
		"based-on-original-patch-by":             {"Influenced-by"},
		"based-on-patch-by":                      {"Influenced-by"},
		"based-on-patch-from":                    {"Influenced-by"},
		"based-on-patches-by":                    {"Influenced-by"},
		"based-on-similar-patches-by":            {"Influenced-by"},
		"based-on-suggestion-from":               {"Influenced-by"},
		"based-on-text-by":                       {"Influenced-by"},
		"based-on-the-original-screenplay-by":    {"Influenced-by"},
		"based-on-the-true-story-by":             {"Influenced-by"},
		"based-on-work-by":                       {"Influenced-by"},
		"based-on-work-from":                     {"Influenced-by"},
		"belatedly-acked-by":                     {"Reviewed-by"},
		"bisected-and-acked-by":                  {"Reviewed-by"},
		"bisected-and-analyzed-by":               {"Reviewed-by"},
		"bisected-and-reported-by":               {"Reported-by"},
		"bisected-and-tested-by":                 {"Reported-by", "Tested-by"},
		"bisected-by":                            {"Reviewed-by"},
		"bisected-reported-and-tested-by":        {"Reviewed-by", "Tested-by"},
		"bitten-by-and-tested-by":                {"Reviewed-by", "Tested-by"},
		"bitterly-acked-by":                      {"Reviewed-by"},
		"blame-taken-by":                         {"Informed-by"},
		"bonus-points-awarded-by":                {"Reviewed-by"},
		"boot-tested-by":                         {"Tested-by"},
		"brainstormed-with":                      {"Influenced-by"},
		"broken-by":                              {"Informed-by"},
		"bug-actually-spotted-by":                {"Reported-by"},
		"bug-fixed-by":                           {"Resolved-by"},
		"bug-found-by":                           {"Reported-by"},
		"bug-identified-by":                      {"Reported-by"},
		"bug-reported-by":                        {"Reported-by"},
		"bug-spotted-by":                         {"Reported-by"},
		"build-fixes-from":                       {"Resolved-by"},
		"build-tested-by":                        {"Tested-by"},
		"build-testing-by":                       {"Tested-by"},
		"catched-by-and-rightfully-ranted-at-by": {"Reported-by"},
		"caught-by":                              {"Reported-by"},
		"cause-discovered-by":                    {"Reported-by"},
		"cautiously-acked-by":                    {"Reviewed-by"},
		"cc":                                     {"Informed-by"},
		"celebrated-by":                          {"Reviewed-by"},
		"changelog-cribbed-from":                 {"Influenced-by"},
		"changelog-heavily-inspired-by":          {"Influenced-by"},
		"chucked-on-by":                          {"Reviewed-by"},
		"cked-by":                                {"Reviewed-by"},
		"cleaned-up-by":                          {"Co-authored-by"},
		"cleanups-from":                          {"Co-authored-by"},
		"co-author":                              {"Co-authored-by"},
		"co-authored":                            {"Co-authored-by"},
		"co-authored-by":                         {"Co-authored-by"},
		"co-debugged-by":                         {"Co-authored-by"},
		"co-developed-by":                        {"Co-authored-by"},
		"co-developed-with":                      {"Co-authored-by"},
		"committed":                              {"Committed-by"},
		"committed-by":                           {"Co-authored-by", "Committed-by"},
		"compile-tested-by":                      {"Tested-by"},
		"compiled-by":                            {"Tested-by"},
		"compiled-tested-by":                     {"Tested-by"},
		"complained-about-by":                    {"Reported-by"},
		"conceptually-acked-by":                  {"Reviewed-by"},
		"confirmed-by":                           {"Reviewed-by"},
		"confirms-rustys-story-ends-the-same-by": {"Reviewed-by"},
		"contributors":                           {"Co-authored-by"},
		"credit":                                 {"Co-authored-by"},
		"credit-to":                              {"Co-authored-by"},
		"credits-by":                             {"Reviewed-by"},
		"csigned-off-by":                         {"Co-authored-by"},
		"cut-and-paste-bug-by":                   {"Reported-by"},
		"debuged-by":                             {"Tested-by"},
		"debugged-and-acked-by":                  {"Reviewed-by"},
		"debugged-and-analyzed-by":               {"Reviewed-by", "Tested-by"},
		"debugged-and-tested-by":                 {"Reviewed-by", "Tested-by"},
		"debugged-by":                            {"Tested-by"},
		"deciphered-by":                          {"Tested-by"},
		"decoded-by":                             {"Tested-by"},
		"delightedly-acked-by":                   {"Reviewed-by"},
		"demanded-by":                            {"Reported-by"},
		"derived-from-code-by":                   {"Co-authored-by"},
		"designed-by":                            {"Influenced-by"},
		"diagnoised-by":                          {"Tested-by"},
		"diagnosed-and-reported-by":              {"Reported-by"},
		"diagnosed-by":                           {"Tested-by"},
		"discovered-and-analyzed-by":             {"Reported-by"},
		"discovered-by":                          {"Reported-by"},
		"discussed-with":                         {"Co-authored-by"},
		"earlier-version-tested-by":              {"Tested-by"},
		"embarrassingly-acked-by":                {"Reviewed-by"},
		"emphatically-acked-by":                  {"Reviewed-by"},
		"encouraged-by":                          {"Influenced-by"},
		"enthusiastically-acked-by":              {"Reviewed-by"},
		"enthusiastically-supported-by":          {"Reviewed-by"},
		"evaluated-by":                           {"Tested-by"},
		"eventually-typed-in-by":                 {"Reported-by"},
		"eviewed-by":                             {"Reviewed-by"},
		"explained-by":                           {"Influenced-by"},
		"fairly-blamed-by":                       {"Reported-by"},
		"fine-by-me":                             {"Reviewed-by"},
		"finished-by":                            {"Co-authored-by"},
		"fix-creation-mandated-by":               {"Resolved-by"},
		"fix-proposed-by":                        {"Resolved-by"},
		"fix-suggested-by":                       {"Resolved-by"},
		"fixed-by":                               {"Resolved-by"},
		"fixes-from":                             {"Resolved-by"},
		"forwarded-by":                           {"Informed-by"},
		"found-by":                               {"Reported-by"},
		"found-ok-by":                            {"Tested-by"},
		"from":                                   {"Informed-by"},
		"grudgingly-acked-by":                    {"Reviewed-by"},
		"grumpily-reviewed-by":                   {"Reviewed-by"},
		"guess-its-ok-by":                        {"Reviewed-by"},
		"hella-acked-by":                         {"Reviewed-by"},
		"helped-by":                              {"Co-authored-by"},
		"helped-out-by":                          {"Co-authored-by"},
		"hinted-by":                              {"Influenced-by"},
		"historical-research-by":                 {"Co-authored-by"},
		"humbly-acked-by":                        {"Reviewed-by"},
		"i-dont-see-any-problems-with-it":        {"Reviewed-by"},
		"idea-by":                                {"Influenced-by"},
		"idea-from":                              {"Influenced-by"},
		"identified-by":                          {"Reported-by"},
		"improved-by":                            {"Influenced-by"},
		"improvements-by":                        {"Influenced-by"},
		"includes-changes-by":                    {"Influenced-by"},
		"initial-analysis-by":                    {"Co-authored-by"},
		"initial-author":                         {"Co-authored-by"},
		"initial-fix-by":                         {"Resolved-by"},
		"initial-patch-by":                       {"Co-authored-by"},
		"initial-work-by":                        {"Co-authored-by"},
		"inspired-by":                            {"Influenced-by"},
		"inspired-by-patch-from":                 {"Influenced-by"},
		"intermittently-reported-by":             {"Reported-by"},
		"investigated-by":                        {"Tested-by"},
		"lightly-tested-by":                      {"Tested-by"},
		"liked-by":                               {"Reviewed-by"},
		"list-usage-fixed-by":                    {"Resolved-by"},
		"looked-over-by":                         {"Reviewed-by"},
		"looks-good-to":                          {"Reviewed-by"},
		"looks-great-to":                         {"Reviewed-by"},
		"looks-ok-by":                            {"Reviewed-by"},
		"looks-okay-to":                          {"Reviewed-by"},
		"looks-reasonable-to":                    {"Reviewed-by"},
		"makes-sense-to":                         {"Reviewed-by"},
		"makes-sparse-happy":                     {"Reviewed-by"},
		"maybe-reported-by":                      {"Reported-by"},
		"mentored-by":                            {"Influenced-by"},
		"modified-and-reviewed-by":               {"Reviewed-by"},
		"modified-by":                            {"Co-authored-by"},
		"more-or-less-tested-by":                 {"Tested-by"},
		"most-definitely-acked-by":               {"Reviewed-by"},
		"mostly-acked-by":                        {"Reviewed-by"},
		"much-requested-by":                      {"Reported-by"},
		"nacked-by":                              {"Reviewed-by"},
		"naked-by":                               {"Reviewed-by"},
		"narrowed-down-by":                       {"Reviewed-by"},
		"niced-by":                               {"Reviewed-by"},
		"no-objection-from-me-by":                {"Reviewed-by"},
		"no-problems-with":                       {"Reviewed-by"},
		"not-nacked-by":                          {"Reviewed-by"},
		"noted-by":                               {"Reviewed-by"},
		"noticed-and-acked-by":                   {"Reviewed-by"},
		"noticed-by":                             {"Reviewed-by"},
		"okay-ished-by":                          {"Reviewed-by"},
		"oked-to-go-through-tracing-tree-by":     {"Reviewed-by"},
		"once-upon-a-time-reviewed-by":           {"Reviewed-by"},
		"original-author":                        {"Co-authored-by"},
		"original-by":                            {"Co-authored-by"},
		"original-from":                          {"Co-authored-by"},
		"original-idea-and-signed-off-by":        {"Co-authored-by"},
		"original-idea-by":                       {"Influenced-by"},
		"original-patch-acked-by":                {"Reviewed-by"},
		"original-patch-by":                      {"Co-authored-by"},
		"original-signed-off-by":                 {"Co-authored-by"},
		"original-version-by":                    {"Co-authored-by"},
		"originalauthor":                         {"Co-authored-by"},
		"originally-by":                          {"Co-authored-by"},
		"originally-from":                        {"Co-authored-by"},
		"originally-suggested-by":                {"Influenced-by"},
		"originally-written-by":                  {"Co-authored-by"},
		"origionally-authored-by":                {"Co-authored-by"},
		"origionally-signed-off-by":              {"Co-authored-by"},
		"partially-reviewed-by":                  {"Reviewed-by"},
		"partially-tested-by":                    {"Tested-by"},
		"partly-suggested-by":                    {"Co-authored-by"},
		"patch-by":                               {"Co-authored-by"},
		"patch-fixed-up-by":                      {"Resolved-by"},
		"patch-from":                             {"Co-authored-by"},
		"patch-inspired-by":                      {"Influenced-by"},
		"patch-originally-by":                    {"Co-authored-by"},
		"patch-updated-by":                       {"Co-authored-by"},
		"patiently-pointed-out-by":               {"Reported-by"},
		"pattern-pointed-out-by":                 {"Influenced-by"},
		"performance-tested-by":                  {"Tested-by"},
		"pinpointed-by":                          {"Reported-by"},
		"pointed-at-by":                          {"Reported-by"},
		"pointed-out-and-tested-by":              {"Reported-by", "Tested-by"},
		"proposed-by":                            {"Reported-by"},
		"pushed-by":                              {"Co-authored-by"},
		"ranted-by":                              {"Reported-by"},
		"re-reported-by":                         {"Reported-by"},
		"reasoning-sounds-sane-to":               {"Reviewed-by"},
		"recalls-having-tested-once-upon-a-time-by": {"Tested-by"},
		"received-from":                                  {"Informed-by"},
		"recommended-by":                                 {"Reviewed-by"},
		"reivewed-by":                                    {"Reviewed-by"},
		"reluctantly-acked-by":                           {"Reviewed-by"},
		"repored-and-bisected-by":                        {"Reported-by"},
		"reporetd-by":                                    {"Reported-by"},
		"reporeted-and-tested-by":                        {"Reported-by", "Tested-by"},
		"report-by":                                      {"Reported-by"},
		"reportded-by":                                   {"Reported-by"},
		"reported":                                       {"Reported-by"},
		"reported--and-debugged-by":                      {"Reported-by", "Tested-by"},
		"reported-acked-and-tested-by":                   {"Reported-by", "Tested-by"},
		"reported-analyzed-and-tested-by":                {"Reported-by"},
		"reported-and-acked-by":                          {"Reviewed-by"},
		"reported-and-bisected-and-tested-by":            {"Reviewed-by", "Tested-by"},
		"reported-and-bisected-by":                       {"Reported-by"},
		"reported-and-reviewed-and-tested-by":            {"Reviewed-by", "Tested-by"},
		"reported-and-root-caused-by":                    {"Reported-by"},
		"reported-and-suggested-by":                      {"Reported-by"},
		"reported-and-test-by":                           {"Reported-by"},
		"reported-and-tested-by":                         {"Tested-by"},
		"reported-any-tested-by":                         {"Tested-by"},
		"reported-bisected-and-tested-by":                {"Reported-by", "Tested-by"},
		"reported-bisected-and-tested-by-the-invaluable": {"Reported-by", "Tested-by"},
		"reported-bisected-tested-by":                    {"Reported-by", "Tested-by"},
		"reported-bistected-and-tested-by":               {"Reported-by", "Tested-by"},
		"reported-by":                                    {"Reported-by"},
		"reported-by-and-tested-by":                      {"Reported-by", "Tested-by"},
		"reported-by-tested-by":                          {"Tested-by"},
		"reported-by-with-patch":                         {"Reported-by"},
		"reported-debuged-tested-acked-by":               {"Tested-by"},
		"reported-off-by":                                {"Reported-by"},
		"reported-requested-and-tested-by":               {"Reported-by", "Tested-by"},
		"reported-reviewed-and-acked-by":                 {"Reviewed-by"},
		"reported-tested-and-acked-by":                   {"Reviewed-by", "Tested-by"},
		"reported-tested-and-bisected-by":                {"Reported-by", "Tested-by"},
		"reported-tested-and-fixed-by":                   {"Co-authored-by", "Reported-by", "Tested-by"},
		"reported-tested-by":                             {"Tested-by"},
		"reported_by":                                    {"Reported-by"},
		"reportedy-and-tested-by":                        {"Reported-by", "Tested-by"},
		"reproduced-by":                                  {"Tested-by"},
		"requested-and-acked-by":                         {"Reviewed-by"},
		"requested-and-tested-by":                        {"Tested-by"},
		"requested-by":                                   {"Reported-by"},
		"researched-with":                                {"Co-authored-by"},
		"reveiewed-by":                                   {"Reviewed-by"},
		"review-by":                                      {"Reviewed-by"},
		"reviewd-by":                                     {"Reviewed-by"},
		"reviewed":                                       {"Reviewed-by"},
		"reviewed-and-tested-by":                         {"Reviewed-by", "Tested-by"},
		"reviewed-and-wanted-by":                         {"Reviewed-by"},
		"reviewed-by":                                    {"Reviewed-by"},
		"reviewed-off-by":                                {"Reviewed-by"},
		"reviewed–by":                                    {"Reviewed-by"},
		"reviewer":                                       {"Reviewed-by"},
		"reviewws-by":                                    {"Reviewed-by"},
		"root-cause-analysis-by":                         {"Reported-by"},
		"root-cause-found-by":                            {"Reported-by"},
		"seconded-by":                                    {"Reviewed-by"},
		"seems-ok":                                       {"Reviewed-by"},
		"seems-reasonable-to":                            {"Reviewed-by"},
		"sefltests-acked-by":                             {"Reviewed-by"},
		"sent-by":                                        {"Informed-by"},
		"serial-parts-acked-by":                          {"Reviewed-by"},
		"siged-off-by":                                   {"Co-authored-by"},
		"sighed-off-by":                                  {"Co-authored-by"},
		"signed":                                         {"Signed-off-by"},
		"signed-by":                                      {"Signed-off-by"},
		"signed-off":                                     {"Signed-off-by"},
		"signed-off-by":                                  {"Co-authored-by", "Signed-off-by"},
		"singend-off-by":                                 {"Co-authored-by"},
		"slightly-grumpily-acked-by":                     {"Reviewed-by"},
		"smoke-tested-by":                                {"Tested-by"},
		"some-suggestions-by":                            {"Influenced-by"},
		"spotted-by":                                     {"Reported-by"},
		"submitted-by":                                   {"Co-authored-by"},
		"suggested-and-acked-by":                         {"Reviewed-by"},
		"suggested-and-reviewed-by":                      {"Reviewed-by"},
		"suggested-and-tested-by":                        {"Reviewed-by", "Tested-by"},
		"suggested-by":                                   {"Reviewed-by"},
		"tested":                                         {"Tested-by"},
		"tested-and-acked-by":                            {"Tested-by"},
		"tested-and-bugfixed-by":                         {"Resolved-by", "Tested-by"},
		"tested-and-reported-by":                         {"Reported-by", "Tested-by"},
		"tested-by":                                      {"Tested-by"},
		"tested-off":                                     {"Tested-by"},
		"thanks-to":                                      {"Influenced-by", "Informed-by"},
		"to":                                             {"Informed-by"},
		"tracked-by":                                     {"Tested-by"},
		"tracked-down-by":                                {"Tested-by"},
		"was-acked-by":                                   {"Reviewed-by"},
		"weak-reviewed-by":                               {"Reviewed-by"},
		"workflow-found-ok-by":                           {"Reviewed-by"},
		"written-by":                                     {"Reported-by"},
	}
	// GitTrailerOtherAuthors - trailer name to authors map (for all documents)
	GitTrailerOtherAuthors = map[string][2]string{
		"Signed-off-by":  {"authors_signed", "signer"},
		"Co-authored-by": {"authors_co_authored", "co_author"},
		"Tested-by":      {"authors_tested", "tester"},
		"Approved-by":    {"authors_approved", "approver"},
		"Reviewed-by":    {"authors_reviewed", "reviewer"},
		"Reported-by":    {"authors_reported", "reporter"},
		"Informed-by":    {"authors_informed", "informer"},
		"Resolved-by":    {"authors_resolved", "resolver"},
		"Influenced-by":  {"authors_influenced", "influencer"},
	}
	// GitTrailerSameAsAuthor - can a given trailer be the same as the main commit's author?
	GitTrailerSameAsAuthor = map[string]bool{
		"Signed-off-by":  true,
		"Co-authored-by": false,
		"Tested-by":      true,
		"Approved-by":    false,
		"Reviewed-by":    false,
		"Reported-by":    true,
		"Informed-by":    true,
		"Resolved-by":    true,
		"Influenced-by":  true,
	}
	// GitTrailerPPAuthors - trailer name to authors map (for pair programming)
	GitTrailerPPAuthors = map[string]string{"Signed-off-by": "authors_signed_off", "Co-authored-by": "co_authors"}
	// max upstream date
	gMaxUpstreamDt             time.Time
	gMaxUpstreamDtMtx          = &sync.Mutex{}
	cachedCommits              = make(map[string]CommitCache)
	commitsCacheFile           = "commits-cache.csv"
	createdCommits             = make(map[string]bool)
	IsHotRep                   = false
	CommitsByYearCacheFile     = "commits-cache-%s.csv"
	CommitsUpdateCacheFile     = "commits-update-cache.csv"
	CurrentCacheYear           = 1970
	CachedCommitsUpdates       = make(map[string]CommitCache)
	CommitsByYearHalfCacheFile = "commits-cache-%s-%s.csv"
	CurrentCacheYearHalf       = YearFirstHalf
	FirstCommitAt              time.Time
)

// Publisher - for streaming data to Kinesis
type Publisher interface {
	PushEvents(action, source, eventType, subEventType, env string, data []interface{}, endpoint string) (string, error)
}

// RawPLS - programming language summary (all fields as strings)
type RawPLS struct {
	Language string `json:"language"`
	Files    string `json:"files"`
	Blank    string `json:"blank"`
	Comment  string `json:"comment"`
	Code     string `json:"code"`
}

// PLS - programming language summary
type PLS struct {
	Language string `json:"language"`
	Files    int    `json:"files"`
	Blank    int    `json:"blank"`
	Comment  int    `json:"comment"`
	Code     int    `json:"code"`
}

// DSGit - DS implementation for git
type DSGit struct {
	URL              string // git repo URL, for example https://github.com/cncf/devstats
	ReposPath        string // path to store git repo clones, defaults to /tmp/git-repositories
	CachePath        string // path to store gitops results cache, defaults to /tmp/git-cache
	SkipCacheCleanup bool   // skip gitops cache cleanup
	// Flags
	FlagURL              *string
	FlagReposPath        *string
	FlagCachePath        *string
	FlagSkipCacheCleanup *bool
	FlagStream           *string
	FlagSourceID         *string
	FlagRepositorySource *string
	// Non-config variables
	RepoName        string // repo name
	Loc             int    // lines of code as reported by GitOpsCommand
	Pls             []PLS  // programming language suppary as reported by GitOpsCommand
	StatsDt         time.Time
	GitPath         string                            // path to git repo clone
	LineScanner     *bufio.Scanner                    // line scanner for git log
	CurrLine        int                               // current line in git log
	ParseState      int                               // 0-init, 1-commit, 2-header, 3-message, 4-file
	Commit          map[string]interface{}            // current parsed commit
	CommitFiles     map[string]map[string]interface{} // current commit's files
	RecentLines     []string                          // recent commit lines
	OrphanedCommits []string                          // orphaned commits SHAs
	OrphanedMap     map[string]struct{}               // orphaned commits SHAs
	DefaultBranch   string                            // default branch name, example: master, main
	Branches        map[string]struct{}               // all branches
	CurrentSHA      string                            // SHA of currently processing commit
	// PairProgramming mode
	PairProgramming bool
	// CommitsHash is a map of commit hashes for each repo
	CommitsHash map[string]map[string]struct{}
	// Publisher & stream
	Publisher
	Stream string // stream to publish the data
	Logger logger.Logger
	// SourceID: the optional external source identifier (such as the repo ID from github/gitlab, or gerrit project slug)
	// this field is required for github, gitlab and gerrit. For github and gitlab, this is typically a numeric value
	// converted to a string such as 194341141. For gerrit this is the project (repository) slug.
	SourceID string
	// RepositorySource for example git, github or gerrit
	RepositorySource string
	log              *logrus.Entry
	cacheProvider    cache.Manager
	endpoint         string
	reportProvider   *report.Manager
	auth0Client      *auth0.ClientProvider
	headCommitHash   string
	headLinesOfCode  int
}

// PublisherPushEvents - this is a fake function to test publisher locally
// FIXME: don't use when done implementing
func (j *DSGit) PublisherPushEvents(ev, ori, src, cat, env string, v []interface{}) error {
	data, err := jsoniter.Marshal(v)
	j.log.WithFields(logrus.Fields{"operation": "PublisherPushEvents"}).Errorf("publish[ev=%s ori=%s src=%s cat=%s env=%s]: %d items: %+v -> %v", ev, ori, src, cat, env, len(v), string(data), err)
	return nil
}

// AddPublisher - sets Kinesis publisher
func (j *DSGit) AddPublisher(publisher Publisher) {
	j.Publisher = publisher
}

// AddLogger - adds logger
func (j *DSGit) AddLogger(ctx *shared.Ctx) {
	client, err := elastic.NewClientProvider(&elastic.Params{
		URL:      os.Getenv("ELASTIC_LOG_URL"),
		Password: os.Getenv("ELASTIC_LOG_PASSWORD"),
		Username: os.Getenv("ELASTIC_LOG_USER"),
	})
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "AddLogger"}).Errorf("create elastic provider error: %+v", err)
		return
	}
	logProvider, err := logger.NewLogger(client, os.Getenv("STAGE"))
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "AddLogger"}).Errorf("create logger provider error: %+v", err)
		return
	}
	j.Logger = *logProvider
}

// WriteLog - writes to log
func (j *DSGit) WriteLog(ctx *shared.Ctx, timestamp time.Time, status, message string) error {
	arn, err := aws.GetContainerARN()
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "WriteLog"}).Errorf("getContainerMetadata Error : %+v", err)
		return err
	}
	err = j.Logger.Write(&logger.Log{
		Connector: GitDataSource,
		TaskARN:   arn,
		Configuration: []map[string]string{
			{
				"REPO_URL":    j.URL,
				"repo_slug":   j.endpoint,
				"ProjectSlug": ctx.Project,
			}},
		Status:    status,
		CreatedAt: timestamp,
		Message:   message,
	})
	return err
}

// AddFlags - add git specific flags
func (j *DSGit) AddFlags() {
	j.FlagURL = flag.String("git-url", "", "git repo URL, for example https://github.com/cncf/devstats")
	j.FlagReposPath = flag.String("git-repos-path", GitDefaultReposPath, "path to store git repo clones, defaults to "+GitDefaultReposPath)
	j.FlagCachePath = flag.String("git-cache-path", GitDefaultCachePath, "path to store gitops results cache, defaults to"+GitDefaultCachePath)
	j.FlagSkipCacheCleanup = flag.Bool("git-skip-cache-cleanup", false, "skip gitops cache cleanup")
	j.FlagStream = flag.String("git-stream", GitDefaultStream, "git kinesis stream name, for example PUT-S3-git-commits")
	j.FlagSourceID = flag.String("git-source-id", "", "repository source id")
	j.FlagRepositorySource = flag.String("git-repository-source", "", "repository source for example git, github or gerrit")
}

// ParseArgs - parse git specific environment variables
func (j *DSGit) ParseArgs(ctx *shared.Ctx) (err error) {
	// git URL
	if shared.FlagPassed(ctx, "url") && *j.FlagURL != "" {
		j.URL = strings.TrimSpace(*j.FlagURL)
	}
	if ctx.EnvSet("URL") {
		j.URL = ctx.Env("URL")
	}

	// git repos path
	j.ReposPath = GitDefaultReposPath
	if shared.FlagPassed(ctx, "repos-path") && *j.FlagReposPath != "" {
		j.ReposPath = *j.FlagReposPath
	}
	if ctx.EnvSet("REPOS_PATH") {
		j.ReposPath = ctx.Env("REPOS_PATH")
	}

	// git cache path
	j.CachePath = GitDefaultCachePath
	if shared.FlagPassed(ctx, "cache-path") && *j.FlagCachePath != "" {
		j.CachePath = *j.FlagCachePath
	}
	if ctx.EnvSet("CACHE_PATH") {
		j.CachePath = ctx.Env("CACHE_PATH")
	}

	// git skip cache cleanup
	if shared.FlagPassed(ctx, "skip-cache-cleanup") {
		j.SkipCacheCleanup = *j.FlagSkipCacheCleanup
	}
	skipCacheCleanup, present := ctx.BoolEnvSet("SKIP_CACHE_CLEANUP")
	if present {
		j.SkipCacheCleanup = skipCacheCleanup
	}

	// git Kinesis stream
	j.Stream = GitDefaultStream
	if shared.FlagPassed(ctx, "stream") {
		j.Stream = *j.FlagStream
	}
	if ctx.EnvSet("STREAM") {
		j.Stream = ctx.Env("STREAM")
	}

	// Some extra initializations
	// NOTE: We enable pair programming by default
	j.PairProgramming = true
	j.CommitsHash = make(map[string]map[string]struct{})
	j.OrphanedMap = make(map[string]struct{})

	// git repository sourceID
	if shared.FlagPassed(ctx, "source-id") {
		j.SourceID = strings.TrimSpace(*j.FlagSourceID)
	}

	// git repository source
	if shared.FlagPassed(ctx, "repository-source") {
		j.RepositorySource = strings.TrimSpace(*j.FlagRepositorySource)
	}

	return
}

// Validate - is current DS configuration OK?
func (j *DSGit) Validate() (err error) {
	url := strings.TrimSpace(j.URL)
	if strings.HasSuffix(url, "/") {
		url = url[:len(url)-1]
	}
	ary := strings.Split(url, "/")
	j.RepoName = ary[len(ary)-1]
	if j.RepoName == "" {
		err = fmt.Errorf("Repo name must be set")
		return
	}
	j.ReposPath = os.ExpandEnv(j.ReposPath)
	if strings.HasSuffix(j.ReposPath, "/") {
		j.ReposPath = j.ReposPath[:len(j.ReposPath)-1]
	}
	j.CachePath = os.ExpandEnv(j.CachePath)
	if strings.HasSuffix(j.CachePath, "/") {
		j.CachePath = j.CachePath[:len(j.CachePath)-1]
	}
	if j.RepositorySource == "" {
		err = fmt.Errorf("repository source must be set, eg: git, github, gerrit")
		return
	}
	return
}

// Init - initialize git data source
func (j *DSGit) Init(ctx *shared.Ctx) (err error) {
	shared.NoSSLVerify()
	ctx.InitEnv("git")
	j.AddFlags()
	ctx.Init()
	ctx.PackSize = PackSize
	err = j.ParseArgs(ctx)
	if err != nil {
		return
	}
	err = j.Validate()
	if err != nil {
		return
	}
	if ctx.Debug > 1 {
		m := &git.Commit{}
		shared.Printf("git: %+v\nshared context: %s\nModel: %+v\n", j, ctx.Info(), m)
	}
	if j.Stream != "" {
		sess, err := session.NewSession()
		if err != nil {
			return err
		}
		s3Client := s3.New(sess)
		objectStore := datalake.NewS3ObjectStore(s3Client)
		datalakeClient := datalake.NewStoreClient(&objectStore)
		j.AddPublisher(&datalakeClient)
	}
	j.AddLogger(ctx)
	return
}

// GetCommitURL - return git commit URL for a given path and SHA
func (j *DSGit) GetCommitURL(origin, hash string) (string, string) {
	if strings.HasPrefix(origin, "git://") {
		return strings.Replace(origin, "git://", "http://", 1) + "/commit/?id=" + hash, "git"
	} else if strings.HasPrefix(origin, "http://git.") || strings.HasPrefix(origin, "https://git.") {
		return origin + "/commit/?id=" + hash, "git"
	} else if strings.Contains(origin, "github.com") {
		return origin + "/commit/" + hash, "github"
	} else if strings.Contains(origin, "gitlab.com") {
		return origin + "/-/commit/" + hash, "gitlab"
	} else if strings.Contains(origin, "bitbucket.org") {
		return origin + "/commits/" + hash, "bitbucket"
	} else if strings.Contains(origin, "gerrit") || strings.Contains(origin, "review") {
		u, err := url.Parse(origin)
		if err != nil {
			j.log.WithFields(logrus.Fields{"operation": "GetCommitURL"}).Error(fmt.Errorf("cannot parse git commit origin: '%s'", origin))
			return origin + "/" + hash, "unknown"
		}
		baseURL := u.Scheme + "://" + u.Host
		vURL := "gitweb"
		if strings.Contains(u.Path, "/gerrit/") {
			vURL = "gerrit/gitweb"
		} else if strings.Contains(u.Path, "/r/") {
			vURL = "r/gitweb"
		}
		project := strings.Replace(u.Path, "/gerrit/", "", -1)
		project = strings.Replace(project, "/r/", "", -1)
		project = strings.TrimLeft(project, "/")
		if !strings.HasSuffix(project, ".git") {
			project += ".git"
		}
		projectURL := "p=" + project
		typeURL := "a=commit"
		hashURL := "h=" + hash
		return baseURL + "/" + vURL + "?" + projectURL + ";" + typeURL + ";" + hashURL, "gerrit"
	} else if strings.Contains(origin, "git.") && (!strings.Contains(origin, "gerrit") || !strings.Contains(origin, "review")) {
		return origin + "/commit/?id=" + hash, "unknown"
	}
	return origin + "/" + hash, "unknown"
}

// GetRepoShortURL - return git commit URL for a given path and SHA
func (j *DSGit) GetRepoShortURL(origin string) (repoShortName string) {
	lastSlashItem := func(arg string) string {
		arg = strings.TrimSuffix(arg, "/")
		arr := strings.Split(arg, "/")
		lArr := len(arr)
		if lArr > 1 {
			return arr[lArr-1]
		}
		return arg
	}
	if strings.Contains(origin, "/github.com/") {
		// https://github.com/org/repo.git --> repo
		arg := strings.TrimSuffix(origin, ".git")
		repoShortName = lastSlashItem(arg)
		return
	} else if strings.Contains(origin, "/gerrit.") {
		// https://gerrit.xyz/r/org/repo -> repo
		repoShortName = lastSlashItem(origin)
		return
	} else if strings.Contains(origin, "/gitlab.com") {
		// https://gitlab.com/org/repo -> repo
		repoShortName = lastSlashItem(origin)
		return
	} else if strings.Contains(origin, "/bitbucket.org/") {
		// https://bitbucket.org/org/repo.git/src/
		arg := strings.TrimSuffix(origin, "/")
		arg = strings.TrimSuffix(arg, "/src")
		arg = strings.TrimSuffix(arg, ".git")
		repoShortName = lastSlashItem(arg)
		return
	}
	// Fall back
	repoShortName = lastSlashItem(origin)
	return
}

// GetOtherTrailersAuthors - get others authors - from other trailers fields (mostly for korg)
// This works on a raw document
func (j *DSGit) GetOtherTrailersAuthors(ctx *shared.Ctx, doc interface{}) (othersMap map[string]map[[2]string]struct{}) {
	// "Signed-off-by":  {"authors_signed", "signer"},
	commitAuthor := ""
	for otherKey, otherRichKey := range GitTrailerOtherAuthors {
		iothers, ok := shared.Dig(doc, []string{"data", otherKey}, false, true)
		if ok {
			sameAsAuthorAllowed, _ := GitTrailerSameAsAuthor[otherKey]
			if !sameAsAuthorAllowed {
				if commitAuthor == "" {
					iCommitAuthor, _ := shared.Dig(doc, []string{"data", "Author"}, true, false)
					commitAuthor = strings.TrimSpace(iCommitAuthor.(string))
					if ctx.Debug > 1 {
						j.log.WithFields(logrus.Fields{"operation": "GetOtherTrailersAuthors"}).Debugf("trailers type %s cannot have the same authors as commit's author %s, checking this", otherKey, commitAuthor)
					}
				}
			}
			others, _ := iothers.([]interface{})
			if ctx.Debug > 1 {
				j.log.WithFields(logrus.Fields{"operation": "GetOtherTrailersAuthors"}).Debugf("other trailers %s -> %s: %s", otherKey, otherRichKey, others)
			}
			if othersMap == nil {
				othersMap = make(map[string]map[[2]string]struct{})
			}
			for _, iOther := range others {
				other := strings.TrimSpace(iOther.(string))
				if !sameAsAuthorAllowed && other == commitAuthor {
					if ctx.Debug > 1 {
						j.log.WithFields(logrus.Fields{"operation": "GetOtherTrailersAuthors"}).Debugf("trailer %s is the same as commit's author, and this isn't allowed for %s trailers, skipping", other, otherKey)
					}
					continue
				}
				_, ok := othersMap[other]
				if !ok {
					othersMap[other] = map[[2]string]struct{}{}
				}
				othersMap[other][otherRichKey] = struct{}{}
			}
		}
	}
	return
}

// IdentityFromGitAuthor - construct identity from git author
func (j *DSGit) IdentityFromGitAuthor(ctx *shared.Ctx, author string) (identity [3]string) {
	fields := strings.Split(author, "<")
	name := strings.TrimSpace(fields[0])
	email := ""
	if len(fields) > 1 {
		fields2 := strings.Split(fields[1], ">")
		email = strings.TrimSpace(fields2[0])
	}
	// We don't attempt to transform email in anyw ay in V2, we just check if this is a correct email (not even checking the domain)
	if email != "" {
		valid, _ := shared.IsValidEmail(email, false, false)
		if !valid {
			email = ""
		}
	}
	identity = [3]string{name, "", email}
	return
}

// EnrichItem - return rich item from raw item for a given author type
func (j *DSGit) EnrichItem(ctx *shared.Ctx, item map[string]interface{}) (rich map[string]interface{}, err error) {
	rich = make(map[string]interface{})
	for _, field := range shared.RawFields {
		v, _ := item[field]
		rich[field] = v
	}
	commit, ok := item["data"].(map[string]interface{})
	if !ok {
		err = fmt.Errorf("missing data field in item %+v", shared.DumpKeys(item))
		return
	}
	iAuthorDate, _ := shared.Dig(commit, []string{"AuthorDate"}, true, false)
	sAuthorDate, _ := iAuthorDate.(string)
	authorDate, authorDateTz, authorTz, ok := shared.ParseDateWithTz(sAuthorDate)
	if !ok {
		err = fmt.Errorf("cannot parse author date from %v", iAuthorDate)
		return
	}
	clocCount, _ := shared.Dig(commit, []string{"cloc_count"}, false, true)
	rich["cloc_count"] = clocCount
	authorLocation := time.FixedZone(fmt.Sprintf("UTC%v", authorTz), int(authorTz)*60*60)
	authorLocalDate := time.Date(authorDate.Year(), authorDate.Month(), authorDate.Day(), authorDate.Hour(), authorDate.Minute(), authorDate.Second(), authorDate.Nanosecond(), authorLocation)
	rich["orphaned"] = false
	rich["tz"] = authorTz
	rich["author_date"] = authorDateTz
	rich["author_local_date"] = authorLocalDate.Format(time.RFC3339)
	rich["author_date_weekday"] = int(authorDateTz.Weekday())
	rich["author_date_hour"] = authorDateTz.Hour()
	rich["utc_author"] = authorDate
	rich["utc_author_date_weekday"] = int(authorDate.Weekday())
	rich["utc_author_date_hour"] = authorDate.Hour()
	iCommitDate, _ := shared.Dig(commit, []string{"CommitDate"}, true, false)
	sCommitDate, _ := iCommitDate.(string)
	commitDate, commitDateTz, commitTz, ok := shared.ParseDateWithTz(sCommitDate)
	if !ok {
		err = fmt.Errorf("cannot parse commit date from %v", iAuthorDate)
		return
	}

	committerLocation := time.FixedZone(fmt.Sprintf("UTC%v", commitTz), int(commitTz)*60*60)
	committerLocationDate := time.Date(commitDate.Year(), commitDate.Month(), commitDate.Day(), commitDate.Hour(), commitDate.Minute(), commitDate.Second(), commitDate.Nanosecond(), committerLocation)
	rich["commit_tz"] = commitTz
	rich["commit_date"] = commitDateTz
	rich["commit_local_date"] = committerLocationDate.Format(time.RFC3339)
	rich["commit_date_weekday"] = int(commitDateTz.Weekday())
	rich["commit_date_hour"] = commitDateTz.Hour()
	rich["utc_commit"] = commitDate
	rich["utc_commit_date_weekday"] = int(commitDate.Weekday())
	rich["utc_commit_date_hour"] = commitDate.Hour()
	message, ok := shared.Dig(commit, []string{"message"}, false, true)
	if ok {
		msg, _ := message.(string)
		ary := strings.Split(msg, "\n")
		rich["title"] = ary[0]
		rich["message_analyzed"] = msg
		if len(msg) > GitMaxMsgLength {
			msg = msg[:GitMaxMsgLength]
		}
		rich["message"] = msg
	} else {
		rich["message_analyzed"] = nil
		rich["message"] = nil
	}
	iBranch, _ := commit["branch"]
	branch, _ := iBranch.(string)
	rich["branch"] = branch
	rich["default_branch"] = j.DefaultBranch
	rich["is_default_branch"] = j.DefaultBranch == branch
	comm, ok := shared.Dig(commit, []string{"commit"}, false, true)
	var hsh string
	if ok {
		hsh, _ = comm.(string)
		rich["hash"] = hsh
		if len(hsh) > 7 {
			rich["hash_short"] = hsh[:7]
		} else {
			rich["hash_short"] = hsh
		}
	} else {
		rich["hash"] = nil
	}
	iRefs, ok := shared.Dig(commit, []string{"refs"}, false, true)
	if ok {
		refsAry, ok := iRefs.([]interface{})
		if ok {
			tags := []string{}
			for _, iRef := range refsAry {
				ref, _ := iRef.(string)
				if strings.Contains(ref, "tag: ") {
					tags = append(tags, ref)
				}
			}
			rich["commit_tags"] = tags
		}
	}
	rich["parents"], _ = commit["parents"]
	rich["branches"] = []interface{}{}
	dtDiff := float64(commitDate.Sub(authorDate).Seconds()) / 3600.0
	dtDiff = math.Round(dtDiff*100.0) / 100.0
	rich["time_to_commit_hours"] = dtDiff
	iRepoName, _ := shared.Dig(item, []string{"origin"}, true, false)
	repoName, _ := iRepoName.(string)
	origin := repoName
	if strings.HasPrefix(repoName, "http") {
		repoName = shared.AnonymizeURL(repoName)
	}
	rich["repo_name"] = repoName
	nFiles := 0
	linesAdded := int64(0)
	linesRemoved := int64(0)
	fileData := []map[string]interface{}{}
	iFiles, ok := shared.Dig(commit, []string{"files"}, false, true)
	if ok {
		files, ok := iFiles.([]map[string]interface{})
		if ok {
			for _, file := range files {
				action, ok := shared.Dig(file, []string{"action"}, false, true)
				if !ok {
					continue
				}
				nFiles++
				iAdded, ok := shared.Dig(file, []string{"added"}, false, true)
				added, removed, name := 0, 0, ""
				if ok {
					added, _ = strconv.Atoi(fmt.Sprintf("%v", iAdded))
					linesAdded += int64(added)
				}
				iRemoved, ok := shared.Dig(file, []string{"removed"}, false, true)
				if ok {
					//removed, _ := iRemoved.(float64)
					removed, _ = strconv.Atoi(fmt.Sprintf("%v", iRemoved))
					linesRemoved += int64(removed)
				}
				iName, ok := shared.Dig(file, []string{"file"}, false, true)
				if ok {
					name, _ = iName.(string)
				}
				fileData = append(
					fileData,
					map[string]interface{}{
						"action":  action,
						"name":    name,
						"added":   added,
						"removed": removed,
					},
				)
			}
		}
	}
	rich["file_data"] = fileData
	rich["files"] = nFiles
	rich["lines_added"] = linesAdded
	rich["lines_removed"] = linesRemoved
	rich["lines_changed"] = linesAdded + linesRemoved
	doc, _ := shared.Dig(commit, []string{"doc_commit"}, false, true)
	rich["doc_commit"] = doc
	empty, _ := shared.Dig(commit, []string{"empty_commit"}, false, true)
	rich["empty_commit"] = empty
	loc, ok := shared.Dig(commit, []string{"total_lines_of_code"}, false, true)
	if ok {
		rich["total_lines_of_code"] = loc
	} else {
		rich["total_lines_of_code"] = 0
	}
	pls, ok := shared.Dig(commit, []string{"program_language_summary"}, false, true)
	if ok {
		rich["program_language_summary"] = pls
	} else {
		rich["program_language_summary"] = []interface{}{}
	}
	rich["commit_url"], rich["commit_repo_type"] = j.GetCommitURL(origin, hsh)
	rich["repo_short_name"] = j.GetRepoShortURL(origin)
	// Printf("commit_url: %+v\n", rich["commit_url"])
	project, ok := shared.Dig(commit, []string{"project"}, false, true)
	if ok {
		rich["project"] = project
	}
	if strings.Contains(origin, GitHubURL) {
		githubRepo := strings.Replace(origin, GitHubURL, "", -1)
		githubRepo = strings.TrimSuffix(githubRepo, ".git")
		rich["github_repo"] = githubRepo
		rich["url_id"] = githubRepo + "/commit/" + hsh
	}
	// authors, committers (can be set from PP)
	var (
		authorsMap    map[string]struct{}
		committersMap map[string]struct{}
	)
	iAuthors, ok := commit["authors"]
	if ok {
		authorsMap, _ = iAuthors.(map[string]struct{})
	} else {
		authorsMap = map[string]struct{}{}
	}
	iCommitters, ok := commit["committers"]
	if ok {
		committersMap, _ = iCommitters.(map[string]struct{})
	} else {
		committersMap = map[string]struct{}{}
	}
	othersMap := j.GetOtherTrailersAuthors(ctx, item)
	idents := [][3]string{}
	identTypes := []string{}
	otherIdents := map[string][3]string{}
	for authorStr := range authorsMap {
		ident, ok := otherIdents[authorStr]
		if !ok {
			ident = j.IdentityFromGitAuthor(ctx, authorStr)
			otherIdents[authorStr] = ident
		}
		idents = append(idents, ident)
		identTypes = append(identTypes, "author")
	}
	for authorStr := range committersMap {
		ident, ok := otherIdents[authorStr]
		if !ok {
			ident = j.IdentityFromGitAuthor(ctx, authorStr)
			otherIdents[authorStr] = ident
		}
		idents = append(idents, ident)
		identTypes = append(identTypes, "committer")
	}
	for authorStr, roles := range othersMap {
		ident, ok := otherIdents[authorStr]
		if !ok {
			ident = j.IdentityFromGitAuthor(ctx, authorStr)
			otherIdents[authorStr] = ident
		}
		for roleData := range roles {
			roleName := roleData[1]
			idents = append(idents, ident)
			identTypes = append(identTypes, roleName)
		}
	}
	rich["idents"] = idents
	rich["ident_types"] = identTypes
	rich["origin"] = shared.AnonymizeURL(rich["origin"].(string))
	rich["tags"] = ctx.Tags
	rich["commit_url"] = shared.AnonymizeURL(rich["commit_url"].(string))
	rich["type"] = "commit"
	// NOTE: From shared
	rich["metadata__enriched_on"] = time.Now()
	// rich[ProjectSlug] = ctx.ProjectSlug
	// rich["groups"] = ctx.Groups
	return
}

// SetParentCommitFlag - additional operations on already enriched item for pair programming
func (j *DSGit) SetParentCommitFlag(richItem map[string]interface{}) (err error) {
	var repoString string
	if repo, ok := richItem["repo_name"]; ok {
		repoString = fmt.Sprintf("%+v", repo)
		if commit, ok := richItem["hash"]; ok {
			commitString := fmt.Sprintf("%+v", commit)
			if innerMap := j.CommitsHash[repoString]; innerMap == nil {
				j.CommitsHash[repoString] = make(map[string]struct{})
			}
			if _, ok := j.CommitsHash[repoString][commitString]; ok {
				// do nothing because the hash exists in the commits map
				richItem["is_parent_commit"] = 0
				return
			}
			j.CommitsHash[repoString][commitString] = struct{}{}
			richItem["is_parent_commit"] = 1
		}
	}
	return
}

func (j *DSGit) dedupAuthors(inContributors []insights.Contributor) (outContributors []insights.Contributor) {
	authors := make(map[string]struct{})
	for _, contributor := range inContributors {
		if string(contributor.Role) == "author" {
			authors[contributor.Identity.ID] = struct{}{}
		}
	}
	for _, contributor := range inContributors {
		if string(contributor.Role) != "co_author" {
			outContributors = append(outContributors, contributor)
			continue
		}
		_, found := authors[contributor.Identity.ID]
		if !found {
			outContributors = append(outContributors, contributor)
		}
	}
	return
}

// GetModelData - return data in lfx-event-schema format
func (j *DSGit) GetModelData(ctx *shared.Ctx, docs []interface{}) []git.CommitCreatedEvent {
	data := make([]git.CommitCreatedEvent, 0)
	baseEvent := service.BaseEvent{
		Type: CommitCreated,
		CRUDInfo: service.CRUDInfo{
			CreatedBy: GitConnector,
			UpdatedBy: GitConnector,
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		},
	}
	commitBaseEvent := git.CommitBaseEvent{
		Connector:        insights.GitConnector,
		ConnectorVersion: GitBackendVersion,
		Source:           insights.Source(j.RepositorySource),
	}

	repoID, err := repository.GenerateRepositoryID(j.SourceID, j.URL, j.RepositorySource)
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "GetModelData"}).Error(fmt.Errorf("GenerateRepositoryID source id: %s, url: %s, source: %s.error:  %+v", j.SourceID, j.URL, j.RepositorySource, err))
	}
	for _, iDoc := range docs {
		commit := git.Commit{}
		doc, _ := iDoc.(map[string]interface{})
		commit.URL, _ = doc["commit_url"].(string)
		commit.SHA, _ = doc["hash"].(string)
		commit.Branch, _ = doc["branch"].(string)
		commit.DefaultBranch, _ = doc["is_default_branch"].(bool)
		commit.ShortHash, _ = doc["hash_short"].(string)
		commit.DocCommit, _ = doc["doc_commit"].(bool)
		commit.Message, _ = doc["message"].(string)
		_, commit.Orphaned = j.OrphanedMap[commit.SHA]
		commit.ParentSHAs, _ = doc["parents"].([]string)
		commit.AuthoredTimestamp, _ = doc["author_date"].(time.Time)
		commit.AuthoredLocalTimestamp, _ = doc["author_local_date"].(string)
		authoredDt, _ := doc["utc_author"].(time.Time)
		commit.RepositoryURL, _ = doc["origin"].(string)
		commit.RepositoryID = repoID
		commitID, err := git.GenerateCommitID(repoID, commit.SHA)
		if err != nil {
			j.log.WithFields(logrus.Fields{"operation": "GetModelData"}).Error(fmt.Errorf("GenerateCommitID repo id: %s, commit sha: %s.error: %+v", repoID, commit.SHA, err))
		}
		commit.ID = commitID
		commit.CommittedTimestamp, _ = doc["commit_date"].(time.Time)
		commit.CommittedLocalTimestamp, _ = doc["commit_local_date"].(string)
		createdOn := authoredDt
		commit.SyncTimestamp = time.Now()
		commitRoles := []insights.Contributor{}
		identsAry, okIdents := doc["idents"].([][3]string)
		identTypesAry, okIdentTypes := doc["ident_types"].([]string)
		if okIdents && okIdentTypes {
			// In pair programming mode co_author need to have custom weight
			ppCoAuthorWeight := 1.0
			if j.PairProgramming {
				nCoAuthors := 0
				for _, identType := range identTypesAry {
					if identType == "co_author" || identType == "author" {
						nCoAuthors++
					}
				}
				if nCoAuthors > 1 {
					ppCoAuthorWeight /= float64(nCoAuthors)
				}
			}
			for i := range identTypesAry {
				commitRole := insights.Contributor{}
				ident := identsAry[i]
				identType := identTypesAry[i]
				commitRole.Role = insights.Role(identType)
				if j.PairProgramming && (identType == "co_author" || identType == "author") {
					commitRole.Weight = ppCoAuthorWeight
				} else {
					commitRole.Weight = 1.0
				}
				name := ident[0]
				username := ""
				email := ident[2]
				// No identity data postprocessing in V2
				// name, username = shared.PostprocessNameUsername(name, username, email)
				userID, err := user.GenerateIdentity(&j.RepositorySource, &email, &name, &username)
				if err != nil {
					j.log.WithFields(logrus.Fields{"operation": "GetModelData"}).Error(fmt.Errorf("GenerateIdentity source: %s, email: %s, name:%s, username:%s. error: %+v", j.RepositorySource, email, name, username, err))
				}
				commitRole.Identity = user.UserIdentityObjectBase{
					ID:         userID,
					Email:      email,
					Name:       name,
					IsVerified: false,
					Username:   username,
					Source:     j.RepositorySource,
				}
				commitRoles = append(commitRoles, commitRole)
			}
		}
		commit.Contributors = j.dedupAuthors(shared.DedupContributors(commitRoles))
		fileCache := make(map[string]*git.CommitFilesByType)
		fileAry, okFileAry := doc["file_data"].([]map[string]interface{})
		if okFileAry {
			for _, fileData := range fileAry {
				fileName, _ := fileData["name"].(string)
				if fileName == "" {
					continue
				}
				ext := ParseFileExtension(fileName)
				if _, ok := fileCache[ext]; !ok {
					fileCache[ext] = &git.CommitFilesByType{Type: ext}
				}
				obj := fileCache[ext]
				linesAdded, _ := fileData["added"].(int)
				obj.LinesAdded += linesAdded
				linesRemoved, _ := fileData["removed"].(int)
				obj.LinesRemoved += linesRemoved
				action, _ := fileData["action"].(string)
				if action == "M" {
					obj.FilesModified++
				} else if action == "D" {
					obj.FilesDeleted++
				} else {
					obj.FilesCreated++
				}
			}
			commit.Files = make([]git.CommitFilesByType, 0)
			for _, value := range fileCache {
				commit.Files = append(commit.Files, *value)
			}
			if len(commit.Files) > 0 {
				clocCount, _ := doc["cloc_count"].(int)
				if clocCount != 0 {
					commit.Files[len(commit.Files)-1].ActualLinesOfCode = clocCount
				}
			}
		}
		commit.MergeCommit = len(fileAry) == 0
		// Event
		data = append(data, git.CommitCreatedEvent{
			CommitBaseEvent: commitBaseEvent,
			BaseEvent:       baseEvent,
			Payload:         commit,
		})
		gMaxUpstreamDtMtx.Lock()
		if createdOn.After(gMaxUpstreamDt) && createdOn.Before(time.Now().UTC()) {
			gMaxUpstreamDt = createdOn
		}
		gMaxUpstreamDtMtx.Unlock()
	}
	return data
}

// ItemID - return unique identifier for an item
func (j *DSGit) ItemID(item interface{}) string {
	id, ok := item.(map[string]interface{})["commit"].(string)
	if !ok {
		shared.Fatalf("git: ItemID() - cannot extract commit from %+v", shared.DumpKeys(item))
	}
	return id
}

// ItemUpdatedOn - return updated on date for an item
func (j *DSGit) ItemUpdatedOn(item interface{}) time.Time {
	iUpdated, _ := shared.Dig(item, []string{GitCommitDateField}, true, false)
	sUpdated, ok := iUpdated.(string)
	if !ok {
		shared.Fatalf("git: ItemUpdatedOn() - cannot extract %s from %+v", GitCommitDateField, shared.DumpKeys(item))
	}
	updated, _, _, ok := shared.ParseDateWithTz(sUpdated)
	if !ok {
		shared.Fatalf("git: ItemUpdatedOn() - cannot extract %s from %s", GitCommitDateField, sUpdated)
	}
	return updated
}

// AddMetadata - add metadata to the item
func (j *DSGit) AddMetadata(ctx *shared.Ctx, item interface{}) (mItem map[string]interface{}) {
	mItem = make(map[string]interface{})
	origin := j.URL
	tags := ctx.Tags
	if len(tags) == 0 {
		tags = []string{origin}
	}
	commitSHA := j.ItemID(item)
	updatedOn := j.ItemUpdatedOn(item)
	uuid := shared.UUIDNonEmpty(ctx, origin, commitSHA)
	timestamp := time.Now()
	mItem["backend_name"] = ctx.DS
	mItem["backend_version"] = GitBackendVersion
	mItem["timestamp"] = fmt.Sprintf("%.06f", float64(timestamp.UnixNano())/1.0e9)
	mItem["uuid"] = uuid
	mItem["origin"] = origin
	mItem["tags"] = tags
	mItem["offset"] = float64(updatedOn.Unix())
	mItem["category"] = "commit"
	mItem["search_fields"] = make(map[string]interface{})
	shared.FatalOnError(shared.DeepSet(mItem, []string{"search_fields", GitDefaultSearchField}, commitSHA, false))
	mItem["metadata__updated_on"] = shared.ToESDate(updatedOn)
	mItem["metadata__timestamp"] = shared.ToESDate(timestamp)
	// mItem[ProjectSlug] = ctx.ProjectSlug
	if ctx.Debug > 1 {
		j.log.WithFields(logrus.Fields{"operation": "AddMetadata"}).Debugf(fmt.Sprintf("origin: %s, uuid: %s, commit sha: %v, updated n: %v", origin, uuid, commitSHA, updatedOn))
	}
	return
}

// GetGitOps - LOC, lang summary stats
func (j *DSGit) GetGitOps(ctx *shared.Ctx, thrN int) (ch chan error, err error) {
	worker := func(c chan error, url string) (e error) {
		defer func() {
			if c != nil {
				c <- e
			}
		}()
		var (
			sout string
			serr string
		)
		cmdLine := []string{GitOpsCommand, url}
		env := map[string]string{
			"DA_GIT_REPOS_PATH": j.ReposPath,
			"DA_GIT_CACHE_PATH": j.CachePath,
		}
		if j.SkipCacheCleanup {
			env["SKIP_CLEANUP"] = "1"
		}
		sout, serr, e = shared.ExecCommand(ctx, cmdLine, "", env)
		if e != nil {
			if GitOpsFailureFatal {
				j.log.WithFields(logrus.Fields{"operation": "GetGitOps"}).Error(fmt.Errorf("error executing command: %v, error: %v, output: %s, error output: %s", cmdLine, e, sout, serr))
			} else {
				j.log.WithFields(logrus.Fields{"operation": "GetGitOps"}).Warningf("WARNING: error executing command: %v, error: %v, output: %s, error output: %s", cmdLine, e, sout, serr)
				e = nil
			}
			return
		}
		type resultType struct {
			Loc int      `json:"loc"`
			Pls []RawPLS `json:"pls"`
		}
		var data resultType
		e = jsoniter.Unmarshal([]byte(sout), &data)
		if e != nil {
			if GitOpsFailureFatal {
				j.log.WithFields(logrus.Fields{"operation": "GetGitOps"}).Errorf("error unmarshaling from %v", sout)
			} else {
				j.log.WithFields(logrus.Fields{"operation": "GetGitOps"}).Warningf("warning unmarshaling from %v", sout)
				e = nil
			}
			return
		}
		j.StatsDt = time.Now()
		j.Loc = data.Loc
		for _, f := range data.Pls {
			files, _ := strconv.Atoi(f.Files)
			blank, _ := strconv.Atoi(f.Blank)
			comment, _ := strconv.Atoi(f.Comment)
			code, _ := strconv.Atoi(f.Code)
			j.Pls = append(
				j.Pls,
				PLS{
					Language: f.Language,
					Files:    files,
					Blank:    blank,
					Comment:  comment,
					Code:     code,
				},
			)
		}
		return
	}
	if thrN <= 1 {
		return nil, worker(nil, j.URL)
	}
	ch = make(chan error)
	go func() { _ = worker(ch, j.URL) }()
	return ch, nil
}

// CreateGitRepo - clone git repo if needed
func (j *DSGit) CreateGitRepo(ctx *shared.Ctx) (err error) {
	info, err := os.Stat(j.GitPath)

	var exists bool
	if !os.IsNotExist(err) {
		if info.IsDir() {
			exists = true
		} else {
			err = fmt.Errorf("%s exists and is a file, not a directory", j.GitPath)
			return
		}
	}
	if !exists {
		if ctx.Debug > 0 {
			j.log.WithFields(logrus.Fields{"operation": "CreateGitRepo"}).Debugf("cloning %s to %s", j.URL, j.GitPath)
		}
		cmdLine := []string{"git", "clone", "--bare", j.URL, j.GitPath}
		env := map[string]string{"LANG": "C"}
		var sout, serr string
		sout, serr, err = shared.ExecCommand(ctx, cmdLine, "", env)
		if err != nil {
			j.log.WithFields(logrus.Fields{"operation": "CreateGitRepo"}).Errorf("error executing command: %v, error: %v, output: %s, output error: %s", cmdLine, err, sout, serr)
			return
		}
		if ctx.Debug > 0 {
			j.log.WithFields(logrus.Fields{"operation": "CreateGitRepo"}).Debugf("cloned %s to %s", j.URL, j.GitPath)
		}
	}
	headPath := j.GitPath + "/HEAD"
	info, err = os.Stat(headPath)
	if os.IsNotExist(err) {
		j.log.WithFields(logrus.Fields{"operation": "CreateGitRepo"}).Errorf("Missing %s file", headPath)
		return
	}
	if info.IsDir() {
		err = fmt.Errorf("%s is a directory, not file", headPath)
	}
	return
}

// UpdateGitRepo - update git repo
func (j *DSGit) UpdateGitRepo(ctx *shared.Ctx) (err error) {
	if ctx.Debug > 0 {
		j.log.WithFields(logrus.Fields{"operation": "CreateGitRepo"}).Debugf("updating repo %s", j.URL)
	}
	cmdLine := []string{"git", "fetch", "origin", "+refs/heads/*:refs/heads/*", "--prune"}
	var sout, serr string
	sout, serr, err = shared.ExecCommand(ctx, cmdLine, j.GitPath, GitDefaultEnv)
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "CreateGitRepo"}).Errorf("error executing %v: %v\n%s\n%s", cmdLine, err, sout, serr)
		return
	}
	if ctx.Debug > 0 {
		j.log.WithFields(logrus.Fields{"operation": "CreateGitRepo"}).Debugf("updated repo %s", j.URL)
	}
	return
}

// GetOrphanedCommits - return data about orphaned commits: commits present in git object storage
// but not present in rev-list - for example squashed commits
func (j *DSGit) GetOrphanedCommits(ctx *shared.Ctx, thrN int) (ch chan error, err error) {
	worker := func(c chan error) (e error) {
		if ctx.Debug > 0 {
			j.log.WithFields(logrus.Fields{"operation": "GetOrphanedCommits"}).Debug("searching for orphaned commits")
		}
		defer func() {
			if c != nil {
				c <- e
			}
		}()
		var (
			sout string
			serr string
		)
		cmdLine := []string{OrphanedCommitsCommand}
		sout, serr, e = shared.ExecCommand(ctx, cmdLine, j.GitPath, GitDefaultEnv)
		if e != nil {
			if OrphanedCommitsFailureFatal {
				j.log.WithFields(logrus.Fields{"operation": "GetOrphanedCommits"}).Errorf("error executing %v: %v\n%s\n%s", cmdLine, e, sout, serr)
			} else {
				j.log.WithFields(logrus.Fields{"operation": "GetOrphanedCommits"}).Warningf("WARNING: error executing %v: %v\n%s\n%s", cmdLine, e, sout, serr)
				e = nil
			}
			return
		}
		ary := strings.Split(sout, " ")
		for _, sha := range ary {
			sha = strings.TrimSpace(sha)
			if sha == "" {
				continue
			}
			j.OrphanedCommits = append(j.OrphanedCommits, sha)
			j.OrphanedMap[sha] = struct{}{}
		}
		j.log.WithFields(logrus.Fields{"operation": "GetOrphanedCommits"}).Infof("found %d orphaned commits", len(j.OrphanedCommits))
		if ctx.Debug > 1 {
			j.log.WithFields(logrus.Fields{"operation": "GetOrphanedCommits"}).Debugf("OrphanedCommits: %+v", j.OrphanedCommits)
		}
		return
	}
	if thrN <= 1 {
		return nil, worker(nil)
	}
	ch = make(chan error)
	go func() { _ = worker(ch) }()
	return ch, nil
}

// GetGitBranches - get default git branch name
func (j *DSGit) GetGitBranches(ctx *shared.Ctx) (err error) {
	if ctx.Debug > 0 {
		j.log.WithFields(logrus.Fields{"operation": "GetGitBranches"}).Debugf("get git branch data from %s", j.GitPath)
	}
	cmdLine := []string{"git", "branch", "-a"}
	var sout, serr string
	sout, serr, err = shared.ExecCommand(ctx, cmdLine, j.GitPath, GitDefaultEnv)
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "GetGitBranches"}).Errorf("error executing %v: %v\n%s\n%s", cmdLine, err, sout, serr)
		return
	}
	if ctx.Debug > 0 {
		j.log.WithFields(logrus.Fields{"operation": "GetGitBranches"}).Debugf("git branch data for %s: %s", j.URL, sout)
	}
	ary := strings.Split(sout, "\n")
	j.Branches = make(map[string]struct{})
	for _, branch := range ary {
		branch := strings.TrimSpace(branch)
		if branch == "" {
			continue
		}
		if ctx.Debug > 1 {
			j.log.WithFields(logrus.Fields{"operation": "GetGitBranches"}).Debugf("branch: '%s'", branch)
		}
		if strings.HasPrefix(branch, "* ") {
			branch = branch[2:]
			if ctx.Debug > 0 {
				j.log.WithFields(logrus.Fields{"operation": "GetGitBranches"}).Debugf("Default branch: '%s'", branch)
			}
			j.DefaultBranch = branch
		}
		j.Branches[branch] = struct{}{}
	}
	if ctx.Debug > 0 {
		j.log.WithFields(logrus.Fields{"operation": "GetGitBranches"}).Debugf("Branches: %v", j.Branches)
	}
	return
}

// ParseGitLog - update git repo
func (j *DSGit) ParseGitLog(ctx *shared.Ctx) (cmd *exec.Cmd, err error) {
	if ctx.Debug > 0 {
		j.log.WithFields(logrus.Fields{"operation": "ParseGitLog"}).Debugf("parsing logs from %s", j.GitPath)
	}
	// Example full command line:
	// LANG=C PAGER='' git log --reverse --topo-order --no-color --decorate --raw --numstat --pretty=fuller --decorate=full --parents -M -C -c
	cmdLine := []string{"git", "log", "--reverse", "--topo-order"}
	cmdLine = append(cmdLine, GitLogOptions...)
	if ctx.DateFrom != nil {
		cmdLine = append(cmdLine, "--since="+shared.ToYMDHMSDate(*ctx.DateFrom))
	}
	if ctx.DateTo != nil {
		cmdLine = append(cmdLine, "--until="+shared.ToYMDHMSDate(*ctx.DateTo))
	}
	var pipe io.ReadCloser
	pipe, cmd, err = shared.ExecCommandPipe(ctx, cmdLine, j.GitPath, GitDefaultEnv)
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "ParseGitLog"}).Errorf("error executing %v: %v", cmdLine, err)
		return
	}
	j.LineScanner = bufio.NewScanner(pipe)
	if ctx.Debug > 0 {
		j.log.WithFields(logrus.Fields{"operation": "ParseGitLog"}).Debugf("created logs scanner %s", j.GitPath)
	}
	return
}

// GetAuthors - parse multiple authors used in pair programming mode
func (j *DSGit) GetAuthors(ctx *shared.Ctx, m map[string]string, n map[string][]string) (authors map[string]struct{}, author string) {
	if ctx.Debug > 1 {
		defer func() {
			j.log.WithFields(logrus.Fields{"operation": "GetAuthors"}).Debugf("GetAuthors(%+v,%+v) -> %+v,%s", m, n, authors, author)
		}()
	}
	if len(m) > 0 {
		authors = make(map[string]struct{})
		email := strings.TrimSpace(m["email"])
		if !strings.Contains(email, "<") || !strings.Contains(email, "@") || !strings.Contains(email, ">") {
			email = ""
		}
		for _, auth := range strings.Split(m["first_authors"], ",") {
			auth = strings.TrimSpace(auth)
			if email != "" && (!strings.Contains(auth, "<") || !strings.Contains(auth, "@") || !strings.Contains(auth, ">")) {
				auth += " " + email
			}
			authors[auth] = struct{}{}
			if author == "" {
				author = auth
			}
		}
		auth := strings.TrimSpace(m["last_author"])
		if email != "" && (!strings.Contains(auth, "<") || !strings.Contains(auth, "@") || !strings.Contains(auth, ">")) {
			auth += " " + email
		}
		authors[auth] = struct{}{}
		if author == "" {
			author = auth
		}
	}
	if len(n) > 0 {
		if authors == nil {
			authors = make(map[string]struct{})
		}
		nEmails := len(n["email"])
		for i, auth := range n["first_authors"] {
			email := ""
			if i < nEmails {
				email = strings.TrimSpace(n["email"][i])
				if !strings.Contains(email, "@") {
					email = ""
				}
			}
			auth = strings.TrimSpace(auth)
			if email != "" && !strings.Contains(auth, "@") {
				auth += " <" + email + ">"
			}
			authors[auth] = struct{}{}
			if author == "" {
				author = auth
			}
		}
	}
	return
}

// GetAuthorsData - extract authors data from a given field (this supports pair programming)
func (j *DSGit) GetAuthorsData(ctx *shared.Ctx, doc interface{}, auth string) (authorsMap map[string]struct{}, firstAuthor string) {
	iauthors, ok := shared.Dig(doc, []string{"data", auth}, false, true)
	if ok {
		authors, _ := iauthors.(string)
		if j.PairProgramming {
			if ctx.Debug > 1 {
				j.log.WithFields(logrus.Fields{"operation": "GetAuthorsData"}).Debugf("pp %s: %s", auth, authors)
			}
			m1 := shared.MatchGroups(GitAuthorsPattern, authors)
			m2 := shared.MatchGroupsArray(GitCoAuthorsPattern, authors)
			if len(m1) > 0 || len(m2) > 0 {
				authorsMap, firstAuthor = j.GetAuthors(ctx, m1, m2)
			}
		}
		if len(authorsMap) == 0 {
			authorsMap = map[string]struct{}{authors: {}}
			firstAuthor = authors
		}
	}
	return
}

// GetOtherPPAuthors - get others authors - possible from fields: Signed-off-by and/or Co-authored-by
func (j *DSGit) GetOtherPPAuthors(ctx *shared.Ctx, doc interface{}) (othersMap map[string]map[string]struct{}) {
	for otherKey := range GitTrailerPPAuthors {
		iothers, ok := shared.Dig(doc, []string{"data", otherKey}, false, true)
		if ok {
			others, _ := iothers.([]interface{})
			if ctx.Debug > 1 {
				j.log.WithFields(logrus.Fields{"operation": "GetOtherPPAuthors"}).Debugf("pp %s: %s", otherKey, others)
			}
			if othersMap == nil {
				othersMap = make(map[string]map[string]struct{})
			}
			for _, iOther := range others {
				other := strings.TrimSpace(iOther.(string))
				_, ok := othersMap[other]
				if !ok {
					othersMap[other] = map[string]struct{}{}
				}
				othersMap[other][otherKey] = struct{}{}
			}
		}
	}
	return
}

// GitEnrichItems - iterate items and enrich them
// items is a current pack of input items
// docs is a pointer to where extracted identities will be stored
func (j *DSGit) GitEnrichItems(ctx *shared.Ctx, thrN int, items []interface{}, docs *[]interface{}, final bool) (err error) {
	j.log.WithFields(logrus.Fields{"operation": "GitEnrichItems"}).Debugf("input processing(%d/%d/%v)", len(items), len(*docs), final)
	outputDocs := func() {
		if len(*docs) > 0 {
			// actual output
			j.log.WithFields(logrus.Fields{"operation": "GitEnrichItems"}).Infof("output processing(%d/%d/%v)", len(items), len(*docs), final)
			data := j.GetModelData(ctx, *docs)
			if j.Publisher != nil {
				formattedData := make([]interface{}, 0)
				updatedData := make([]interface{}, 0)
				commits := make([]CommitCache, 0)
				updateCommits := make([]CommitCache, 0)
				for _, d := range data {
					contentHash, er := createHash(d.Payload)
					if er != nil {
						j.log.WithFields(logrus.Fields{"operation": "GitEnrichItems"}).Errorf("error hash data for commit %s, error %v", d.Payload, err)
						continue
					}
					commitB, err := jsoniter.Marshal(d.Payload)
					if err != nil {
						return
					}
					commitStr := b64.StdEncoding.EncodeToString(commitB)
					hashExist := isHashCreated(contentHash)
					isCreated := isCommitCreated(d.Payload.ID)
					if !hashExist && !isCreated {
						formattedData = append(formattedData, d)
						tStamp := d.Payload.SyncTimestamp.Unix()
						comm := CommitCache{
							Timestamp:      fmt.Sprintf("%v", tStamp),
							EntityID:       d.Payload.ID,
							SourceEntityID: d.Payload.SHA,
							Content:        commitStr,
							Hash:           contentHash,
							CommitDate:     d.Payload.CommittedTimestamp,
						}
						if IsHotRep {
							comm.Content = ""
						}
						commits = append(commits, comm)
						createdCommits[d.Payload.ID] = true
					}
					if isCreated && !hashExist {
						updatedEvent := git.CommitUpdatedEvent{
							CommitBaseEvent: d.CommitBaseEvent,
							BaseEvent: service.BaseEvent{
								Type:     CommitUpdated,
								CRUDInfo: d.BaseEvent.CRUDInfo,
							},
							Payload: d.Payload,
						}
						updatedData = append(updatedData, updatedEvent)
						tStamp := d.Payload.SyncTimestamp.Unix()
						comm := CommitCache{
							Timestamp:      fmt.Sprintf("%v", tStamp),
							EntityID:       d.Payload.ID,
							SourceEntityID: d.Payload.SHA,
							Content:        commitStr,
							Hash:           contentHash,
							CommitDate:     d.Payload.CommittedTimestamp,
						}
						if IsHotRep {
							comm.Content = ""
						}
						updateCommits = append(updateCommits, comm)
					}

				}
				if len(formattedData) > 0 {
					path, err := j.Publisher.PushEvents(CommitCreated, "insights", GitDataSource, "commits", os.Getenv("STAGE"), formattedData, j.endpoint)
					if err != nil {
						j.log.WithFields(logrus.Fields{"operation": "GitEnrichItems"}).Errorf("Error: %+v", err)
						return
					}
					if !IsHotRep {
						if err = j.createCacheFile(commits, path); err != nil {
							return
						}
					} else {
						if err = j.createYearHalfCacheFile(commits, path); err != nil {
							return
						}
					}
				}
				if len(updatedData) > 0 {
					path, err := j.Publisher.PushEvents(CommitUpdated, "insights", GitDataSource, "commits", os.Getenv("STAGE"), updatedData, j.endpoint)
					if err != nil {
						j.log.WithFields(logrus.Fields{"operation": "GitEnrichItems"}).Errorf("Error: %+v", err)
						return
					}
					if !IsHotRep {
						if err = j.createCacheFile(updateCommits, path); err != nil {
							return
						}
					} else {
						if err = j.createUpdateCacheFile(updateCommits, path); err != nil {
							return
						}
					}

				}

			} else {
				var jsonBytes []byte
				jsonBytes, err = jsoniter.Marshal(data)
				if err != nil {
					j.log.WithFields(logrus.Fields{"operation": "GitEnrichItems"}).Errorf("Error: %+v", err)
					return
				}
				j.log.WithFields(logrus.Fields{"operation": "GitEnrichItems"}).Infof("%s", string(jsonBytes))
			}
			*docs = []interface{}{}
			err = j.setLastSync(ctx)
			if err != nil {
				return
			}

			// write report data
			rData := new(ReportData)
			rData.NewCommits = int64(len(data))
			rData.URL = j.URL
			rData.Date = time.Now().UnixNano()

			b, err := jsoniter.Marshal(rData)
			if err != nil {
				return
			}

			err = j.reportProvider.UpdateFileByKey(fmt.Sprintf("%+v-%+v.json", j.endpoint, time.Now().Unix()), b)
			if err != nil {
				return
			}

		}
	}
	if final {
		defer func() {
			outputDocs()
		}()
	}
	// NOTE: non-generic code starts
	var (
		mtx *sync.RWMutex
		ch  chan error
	)
	if thrN > 1 {
		mtx = &sync.RWMutex{}
		ch = make(chan error)
	}
	var getRichItem func(map[string]interface{}) (interface{}, error)
	if j.PairProgramming {
		// PP
		getRichItem = func(doc map[string]interface{}) (rich interface{}, e error) {
			idata, _ := shared.Dig(doc, []string{"data"}, true, false)
			data, _ := idata.(map[string]interface{})
			data["Author-Original"] = data["Author"]
			authorsMap, firstAuthor := j.GetAuthorsData(ctx, doc, "Author")
			if len(authorsMap) > 0 {
				data["authors"] = authorsMap
				data["Author"] = firstAuthor
			}
			committersMap, firstCommitter := j.GetAuthorsData(ctx, doc, "Commit")
			if len(committersMap) > 0 {
				data["committers"] = committersMap
				data["Commit-Original"] = data["Commit"]
				data["Commit"] = firstCommitter
			}
			rich, e = j.EnrichItem(ctx, doc)
			return
		}
	} else {
		// Non PP
		getRichItem = func(doc map[string]interface{}) (rich interface{}, e error) {
			rich, e = j.EnrichItem(ctx, doc)
			return
		}
	}
	nThreads := 0
	procItem := func(c chan error, idx int) (e error) {
		if thrN > 1 {
			mtx.RLock()
		}
		item := items[idx]
		if thrN > 1 {
			mtx.RUnlock()
		}
		defer func() {
			if c != nil {
				c <- e
			}
		}()
		// NOTE: never refer to _source - we no longer use ES
		doc, ok := item.(map[string]interface{})
		if !ok {
			e = fmt.Errorf("Failed to parse document %+v", doc)
			return
		}
		rich, e := getRichItem(doc)
		if e != nil {
			return
		}
		if thrN > 1 {
			mtx.Lock()
		}
		e = j.SetParentCommitFlag(rich.(map[string]interface{}))
		if e != nil {
			if thrN > 1 {
				mtx.Unlock()
			}
			return
		}
		if thrN > 1 {
			mtx.Unlock()
		}
		if thrN > 1 {
			mtx.Lock()
		}
		*docs = append(*docs, rich)
		// NOTE: flush here
		if len(*docs) >= ctx.PackSize {
			outputDocs()
		}
		if thrN > 1 {
			mtx.Unlock()
		}
		return
	}
	if thrN > 1 {
		for i := range items {
			go func(i int) {
				_ = procItem(ch, i)
			}(i)
			nThreads++
			if nThreads == thrN {
				err = <-ch
				if err != nil {
					return
				}
				nThreads--
			}
		}
		for nThreads > 0 {
			err = <-ch
			nThreads--
			if err != nil {
				return
			}
		}
		return
	}
	for i := range items {
		err = procItem(nil, i)
		if err != nil {
			return
		}
	}
	return
}

// GetCommitBranch - get commit branch from refs
func (j *DSGit) GetCommitBranch(ctx *shared.Ctx, refs []string) (branch string) {
	// ref can be:
	// tag: refs/tags/0.9.0
	// refs/heads/DA-2371-prod
	// HEAD -> unicron-add-branches, origin/main, origin/HEAD, main
	// origin/main, origin/HEAD, main
	tag := ""
	for _, ref := range refs {
		isTag := false
		if strings.HasPrefix(ref, "tag: ") {
			isTag = true
			ref = ref[5:]
		}
		ary := strings.Split(ref, " -> ")
		if len(ary) > 0 {
			ref = ary[len(ary)-1]
		}
		ref = strings.Replace(ref, "origin/", "", 1)
		ref = strings.Replace(ref, "refs/heads/", "", 1)
		if isTag == true {
			tag = ref
			continue
		}
		if ref == j.DefaultBranch {
			continue
		}
		branch = ref
	}
	if branch == "" && tag != "" {
		branch = tag
	}
	if branch == "" {
		branch = j.DefaultBranch
	}
	if ctx.Debug > 1 {
		j.log.WithFields(logrus.Fields{"operation": "GetCommitBranch"}).Debugf("Branch: %+v -> %s", refs, branch)
	}
	return
}

// ParseCommit - parse commit
func (j *DSGit) ParseCommit(ctx *shared.Ctx, line string) (parsed bool, err error) {
	m := shared.MatchGroups(GitCommitPattern, line)
	if len(m) == 0 {
		err = fmt.Errorf("expecting commit on line %d: '%s'", j.CurrLine, line)
		return
	}
	j.CurrentSHA = m["commit"]
	parentsAry := []string{}
	refsAry := []string{}
	parents, parentsPresent := m["parents"]
	if parentsPresent && parents != "" {
		parentsAry = strings.Split(strings.TrimSpace(parents), " ")
	}
	refs, refsPresent := m["refs"]
	if refsPresent && refs != "" {
		ary := strings.Split(strings.TrimSpace(refs), ",")
		for _, ref := range ary {
			ref = strings.TrimSpace(ref)
			if ref != "" {
				refsAry = append(refsAry, ref)
			}
		}
	}
	j.Commit = make(map[string]interface{})
	j.Commit["commit"] = j.CurrentSHA
	j.Commit["parents"] = parentsAry
	j.Commit["refs"] = refsAry
	if len(refsAry) > 0 {
		j.Commit["branch"] = j.GetCommitBranch(ctx, refsAry)
	} else {
		j.Commit["branch"] = j.DefaultBranch
	}
	j.CommitFiles = make(map[string]map[string]interface{})
	j.ParseState = GitParseStateHeader
	parsed = true
	return
}

// ExtractPrevFileName - extracts previous file name (before rename/move etc.)
func (*DSGit) ExtractPrevFileName(f string) (res string) {
	i := strings.Index(f, "{")
	j := strings.Index(f, "}")
	if i > -1 && j > -1 {
		k := shared.IndexAt(f, " => ", i)
		if k > -1 {
			prefix := f[:i]
			inner := f[i+1 : k]
			suffix := f[j+1:]
			res = prefix + inner + suffix
		}
	} else if strings.Index(f, " => ") > -1 {
		res = strings.Split(f, " => ")[0]
	} else {
		res = f
	}
	return
}

// BuildCommit - return commit structure from the current parsed object
func (j *DSGit) BuildCommit(ctx *shared.Ctx) (commit map[string]interface{}) {
	if ctx.Debug > 2 {
		defer func() {
			j.log.WithFields(logrus.Fields{"operation": "BuildCommit"}).Debug("built commit %+v", commit)
		}()
	}
	commit = j.Commit
	ks := []string{}
	for k, v := range commit {
		if v == nil {
			ks = append(ks, k)
		}
	}
	for _, k := range ks {
		delete(commit, k)
	}
	files := []map[string]interface{}{}
	sf := []string{}
	doc := false
	for f := range j.CommitFiles {
		sf = append(sf, f)
		if GitDocFilePattern.MatchString(f) {
			doc = true
		}
	}
	sort.Strings(sf)
	for _, f := range sf {
		d := j.CommitFiles[f]
		ks = []string{}
		if ctx.Debug > 1 {
			j.log.WithFields(logrus.Fields{"operation": "BuildCommit"}).Debugf("%s: '%s'->%+v", j.CurrentSHA, f, d)
		}
		for k, v := range d {
			if v == nil {
				ks = append(ks, k)
			}
		}
		if ctx.Debug > 1 {
			j.log.WithFields(logrus.Fields{"operation": "BuildCommit"}).Debugf("%s: delete %+v", j.CurrentSHA, ks)
		}
		for _, k := range ks {
			delete(d, k)
		}
		files = append(files, d)
	}
	commit["files"] = files
	commit["doc_commit"] = doc
	j.Commit = nil
	j.CommitFiles = nil
	return
}

// ParseStats - parse stats line
func (j *DSGit) ParseStats(ctx *shared.Ctx, data map[string]string) {
	fileName := j.ExtractPrevFileName(data["file"])
	if ctx.Debug > 1 {
		j.log.WithFields(logrus.Fields{"operation": "ParseStats"}).Debugf("%s: '%s' --> '%s'", j.CurrentSHA, data["file"], fileName)
	}
	prevData, ok := j.CommitFiles[fileName]
	prevAdded, prevRemoved := 0, 0
	if !ok {
		j.CommitFiles[fileName] = make(map[string]interface{})
		j.CommitFiles[fileName]["file"] = fileName
	} else {
		prevAdded, _ = prevData["added"].(int)
		prevRemoved, _ = prevData["removed"].(int)
	}
	added, _ := strconv.Atoi(data["added"])
	removed, _ := strconv.Atoi(data["removed"])
	j.CommitFiles[fileName]["added"] = prevAdded + added
	j.CommitFiles[fileName]["removed"] = prevRemoved + removed
}

// ParseFile - parse file state
func (j *DSGit) ParseFile(ctx *shared.Ctx, line string) (parsed, empty bool, err error) {
	if line == "" {
		j.ParseState = GitParseStateCommit
		parsed = true
		return
	}
	m := shared.MatchGroups(GitActionPattern, line)
	if len(m) > 0 {
		j.ParseAction(ctx, m)
		parsed = true
		return
	}
	m = shared.MatchGroups(GitStatsPattern, line)
	if len(m) > 0 {
		j.ParseStats(ctx, m)
		parsed = true
		return
	}
	m = shared.MatchGroups(GitCommitPattern, line)
	if len(m) > 0 {
		empty = true
	} else if ctx.Debug > 1 {
		j.log.WithFields(logrus.Fields{"operation": "ParseFile"}).Debugf("invalid file section format, line %d: '%s'", j.CurrLine, line)
	}
	j.ParseState = GitParseStateCommit
	return
}

// ParseHeader - parse header state
func (j *DSGit) ParseHeader(ctx *shared.Ctx, line string) (parsed bool, err error) {
	if line == "" {
		j.ParseState = GitParseStateMessage
		parsed = true
		return
	}
	m := shared.MatchGroups(GitHeaderPattern, line)
	if len(m) == 0 {
		err = fmt.Errorf("invalid header format, line %d: '%s'", j.CurrLine, line)
		return
	}
	// Not too many properties, ES has 1000 fields limit, and each commit can have
	// different properties, so value around 300 should(?) be safe
	if len(j.Commit) < GitMaxCommitProperties {
		if m["name"] != "" {
			j.Commit[m["name"]] = m["value"]
		}
	}
	parsed = true
	return
}

// ParseMessage - parse message state
func (j *DSGit) ParseMessage(ctx *shared.Ctx, line string) (parsed bool, err error) {
	if line == "" {
		j.ParseState = GitParseStateFile
		parsed = true
		return
	}
	m := shared.MatchGroups(GitMessagePattern, line)
	if len(m) == 0 {
		if ctx.Debug > 1 {
			j.log.WithFields(logrus.Fields{"operation": "ParseMessage"}).Debugf("invalid message format, line %d: '%s'", j.CurrLine, line)
		}
		j.ParseState = GitParseStateFile
		return
	}
	msg := m["msg"]
	currMsg, ok := j.Commit["message"]
	if ok {
		sMsg, _ := currMsg.(string)
		j.Commit["message"] = sMsg + "\n" + msg
	} else {
		j.Commit["message"] = msg
	}
	j.ParseTrailer(ctx, msg)
	parsed = true
	return
}

// ParseAction - parse action line
func (j *DSGit) ParseAction(ctx *shared.Ctx, data map[string]string) {
	var (
		modesAry   []string
		indexesAry []string
	)
	modes, modesPresent := data["modes"]
	if modesPresent && modes != "" {
		modesAry = strings.Split(strings.TrimSpace(modes), " ")
	}
	indexes, indexesPresent := data["indexes"]
	if indexesPresent && indexes != "" {
		indexesAry = strings.Split(strings.TrimSpace(indexes), " ")
	}
	fileName := data["file"]
	_, ok := j.CommitFiles[fileName]
	if !ok {
		j.CommitFiles[fileName] = make(map[string]interface{})
	}
	j.CommitFiles[fileName]["modes"] = modesAry
	j.CommitFiles[fileName]["indexes"] = indexesAry
	j.CommitFiles[fileName]["action"] = data["action"]
	j.CommitFiles[fileName]["file"] = fileName
	j.CommitFiles[fileName]["newfile"] = data["newfile"]
}

// ParseFileExtension - return file extension if present
func ParseFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) == 0 {
		return UnknownExtension
	}
	extension := parts[len(parts)-1]
	if extension == "" {
		return UnknownExtension
	}
	return extension
}

// ParseTrailer - parse possible trailer line
func (j *DSGit) ParseTrailer(ctx *shared.Ctx, line string) {
	m := shared.MatchGroups(GitTrailerPattern, line)
	if len(m) == 0 {
		return
	}
	oTrailer := m["name"]
	lTrailer := strings.ToLower(oTrailer)
	trailers, ok := GitAllowedTrailers[lTrailer]
	if !ok {
		if ctx.Debug > 1 {
			j.log.WithFields(logrus.Fields{"operation": "ParseTrailer"}).Debugf("Trailer %s/%s not in the allowed list %v, skipping", oTrailer, lTrailer, GitAllowedTrailers)
		}
		return
	}
	for _, trailer := range trailers {
		ary, ok := j.Commit[trailer]
		if ok {
			if ctx.Debug > 1 {
				j.log.WithFields(logrus.Fields{"operation": "ParseTrailer"}).Debugf("trailer %s -> %s found in '%s'", oTrailer, trailer, line)
			}
			// Trailer can be the same as header value, we still want to have it - with "-Trailer" prefix added
			_, ok = ary.(string)
			if ok {
				trailer += "-Trailer"
				ary2, ok2 := j.Commit[trailer]
				if ok2 {
					if ctx.Debug > 1 {
						j.log.WithFields(logrus.Fields{"operation": "ParseTrailer"}).Debugf("renamed trailer %s -> %s found in '%s'", oTrailer, trailer, line)
					}
					j.Commit[trailer] = append(ary2.([]interface{}), m["value"])
				} else {
					if ctx.Debug > 1 {
						j.log.WithFields(logrus.Fields{"operation": "ParseTrailer"}).Debugf("added renamed trailer %s", trailer)
					}
					j.Commit[trailer] = []interface{}{m["value"]}
				}
			} else {
				j.Commit[trailer] = shared.UniqueStringArray(append(ary.([]interface{}), m["value"]))
				if ctx.Debug > 1 {
					j.log.WithFields(logrus.Fields{"operation": "ParseTrailer"}).Debugf("appended trailer %s -> %s found in '%s'", oTrailer, trailer, line)
				}
			}
		} else {
			j.Commit[trailer] = []interface{}{m["value"]}
		}
	}
}

// ParseInit - parse initial state
func (j *DSGit) ParseInit(ctx *shared.Ctx, line string) (parsed bool, err error) {
	j.ParseState = GitParseStateCommit
	parsed = line == ""
	return
}

// HandleRecentLines - keep last 30 lines, so we can show them on parser error
func (j *DSGit) HandleRecentLines(line string) {
	j.RecentLines = append(j.RecentLines, line)
	l := len(j.RecentLines)
	if l > 30 {
		j.RecentLines = j.RecentLines[1:]
	}
}

func (j *DSGit) BuildCommitMap(comm object.Commit) (map[string]interface{}, error) {
	commit := make(map[string]interface{})
	commit["commit"] = comm.Hash.String()
	parents := make([]string, 0)
	for _, p := range comm.ParentHashes {
		if !p.IsZero() {
			parents = append(parents, p.String())
		}
	}
	commit["parents"] = parents
	commit["refs"] = comm
	commit["branch"] = j.DefaultBranch
	commit["message"] = strings.TrimSpace(comm.Message)
	commit["Author"] = comm.Author.String()
	commit["Commit"] = comm.Committer.String()
	commit["CommitDate"] = comm.Committer.When.Format(time.RFC1123Z)
	commit["AuthorDate"] = comm.Author.When.Format(time.RFC1123Z)
	files := make([]map[string]interface{}, 0)
	states, err := comm.Stats()
	if err != nil {
		return commit, err
	}

	fileStates := make(map[string]object.FileStat)
	doc := false
	for _, s := range states {
		fileStates[s.Name] = s
	}

	allFiles, err := getFilesAction(comm)
	if err != nil {
		return commit, err
	}

	for k, v := range allFiles {
		s, ok := fileStates[k]
		if ok {
			f := make(map[string]interface{})
			f["file"] = s.Name
			f["added"] = s.Addition
			f["removed"] = s.Deletion
			f["action"] = v
			if GitDocFilePattern.MatchString(s.Name) {
				doc = true
			}
			files = append(files, f)
		} else {
			f := make(map[string]interface{})
			f["file"] = k
			f["action"] = v
			if GitDocFilePattern.MatchString(k) {
				doc = true
			}
			files = append(files, f)
		}

	}

	commit["files"] = files
	commit["doc_commit"] = doc
	return commit, nil
}

func getFilesAction(com object.Commit) (map[string]string, error) {
	t, _ := com.Tree()
	toTree := &object.Tree{}
	if com.NumParents() != 0 {
		firstParent, err := com.Parents().Next()
		if err != nil {
			return map[string]string{}, err
		}

		toTree, err = firstParent.Tree()
		if err != nil {
			return map[string]string{}, err
		}
	}
	patch, err := toTree.PatchContext(context.Background(), t)
	if err != nil {
		return map[string]string{}, err
	}

	fs := patch.FilePatches()
	filesAction := make(map[string]string, len(fs))
	for _, fp := range fs {
		from, to := fp.Files()

		if from != nil && to != nil {
			if from.Path() != to.Path() {
				// file is renamed which we ignored
				continue
			}
			filesAction[from.Path()] = "M"
			continue
		}

		if to == nil {
			if from != nil {
				filesAction[from.Path()] = "D"
				continue
			}
		}

		if from == nil {
			if to != nil {
				filesAction[to.Path()] = ""
				continue
			}
		}
	}
	return filesAction, nil
}

// ParseNextCommit - parse next git log commit or report end
func (j *DSGit) ParseNextCommit(ctx *shared.Ctx) (commit map[string]interface{}, ok bool, err error) {
	for j.LineScanner.Scan() {
		j.CurrLine++
		line := strings.TrimRight(j.LineScanner.Text(), "\n")
		if ctx.Debug > 2 {
			j.HandleRecentLines(line)
		}
		if ctx.Debug > 2 {
			j.log.WithFields(logrus.Fields{"operation": "ParseNextCommit"}).Debugf("line %d: '%s'", j.CurrLine, line)
		}
		var (
			parsed bool
			empty  bool
			state  string
		)
		for {
			if ctx.Debug > 2 {
				state = fmt.Sprintf("%v", j.ParseState)
			}
			switch j.ParseState {
			case GitParseStateInit:
				parsed, err = j.ParseInit(ctx, line)
			case GitParseStateCommit:
				parsed, err = j.ParseCommit(ctx, line)
			case GitParseStateHeader:
				parsed, err = j.ParseHeader(ctx, line)
			case GitParseStateMessage:
				parsed, err = j.ParseMessage(ctx, line)
			case GitParseStateFile:
				parsed, empty, err = j.ParseFile(ctx, line)
			default:
				err = fmt.Errorf("unknown parse state:%d", j.ParseState)
			}
			if ctx.Debug > 2 {
				state += fmt.Sprintf(" -> (%v,%v,%v)", j.ParseState, parsed, err)
				j.log.WithFields(logrus.Fields{"operation": "ParseNextCommit"}).Debugf("%s", state)
			}
			if err != nil {
				j.log.WithFields(logrus.Fields{"operation": "ParseNextCommit"}).Errorf("parse next line '%s' error: %v", line, err)
				return
			}
			if j.ParseState == GitParseStateCommit && j.Commit != nil {
				commit = j.BuildCommit(ctx)
				if empty {
					commit["empty_commit"] = true
					parsed, err = j.ParseCommit(ctx, line)
					if !parsed || err != nil {
						j.log.WithFields(logrus.Fields{"operation": "ParseNextCommit"}).Error("failed to parse commit after empty file section")
						return
					}
				}
				ok = true
				return
			}
			if parsed {
				break
			}
		}
	}
	if j.Commit != nil {
		commit = j.BuildCommit(ctx)
		ok = true
	}
	return
}

// Sync - sync git data source
func (j *DSGit) Sync(ctx *shared.Ctx) (err error) {
	thrN := shared.GetThreadsNum(ctx)
	lastSync := os.Getenv("LAST_SYNC")
	if lastSync != "" {
		i, err := strconv.ParseInt(lastSync, 10, 64)
		if err != nil {
			return err
		}
		tm := time.Unix(i, 0).UTC()
		ctx.DateFrom = &tm
	}
	if ctx.DateFrom != nil {
		j.log.WithFields(logrus.Fields{"operation": "Sync"}).Infof("%s fetching from %v (%d threads)", j.URL, ctx.DateFrom, thrN)
	}
	if ctx.DateFrom == nil {
		lastSyncDataB, er := j.cacheProvider.GetLastSyncFile(j.endpoint)
		if er != nil {
			err = er
			return
		}
		var lastSyncData lastSyncFile
		if er = jsoniter.Unmarshal(lastSyncDataB, &lastSyncData); er != nil {
			var cachedLastSync time.Time
			err = jsoniter.Unmarshal(lastSyncDataB, &cachedLastSync)
			if err != nil {
				err = er
				return
			}
			lastSyncData = lastSyncFile{
				LastSync: cachedLastSync,
			}
		}
		ctx.DateFrom = &lastSyncData.LastSync
		if ctx.DateFrom != nil {
			j.log.WithFields(logrus.Fields{"operation": "Sync"}).Infof("%s resuming from %v (%d threads)", j.URL, ctx.DateFrom, thrN)
		}
	}
	j.getCache(lastSync)
	if ctx.DateTo != nil {
		j.log.WithFields(logrus.Fields{"operation": "Sync"}).Infof("%s fetching till %v (%d threads)", j.URL, ctx.DateTo, thrN)
	}
	// NOTE: Non-generic starts here
	var (
		ch            chan error
		allDocs       []interface{}
		allCommits    []interface{}
		allCommitsMtx *sync.Mutex
		escha         []chan error
		eschaMtx      *sync.Mutex
		goch          chan error
		occh          chan error
		waitLOCMtx    *sync.Mutex
	)
	if thrN > 1 {
		ch = make(chan error)
		allCommitsMtx = &sync.Mutex{}
		eschaMtx = &sync.Mutex{}
		waitLOCMtx = &sync.Mutex{}
		goch, _ = j.GetGitOps(ctx, thrN)
	} else {
		_, err = j.GetGitOps(ctx, thrN)
		if err != nil {
			return
		}
	}
	// Do normal git processing, which don't needs gitops yet
	j.GitPath = j.ReposPath + "/" + j.URL + "-git"
	j.GitPath, err = shared.EnsurePath(j.GitPath, true)
	shared.FatalOnError(err)
	if ctx.Debug > 0 {
		j.log.WithFields(logrus.Fields{"operation": "Sync"}).Debugf("path to store git repository: %s", j.GitPath)
	}
	shared.FatalOnError(j.CreateGitRepo(ctx))
	shared.FatalOnError(j.UpdateGitRepo(ctx))
	if thrN > 1 {
		occh, _ = j.GetOrphanedCommits(ctx, thrN)
	} else {
		_, err = j.GetOrphanedCommits(ctx, thrN)
		if err != nil {
			return
		}
	}
	err = j.GetGitBranches(ctx)
	if err != nil {
		return
	}
	var cmd *exec.Cmd
	cmd, err = j.ParseGitLog(ctx)
	if err != nil {
		return
	}

	sourceID := ""
	if j.RepositorySource == "github" {
		sourceID, err = j.getGithubRepoSourceId()
		if err != nil {
			j.log.WithFields(logrus.Fields{"operation": "sync"}).Error(fmt.Errorf("getGithubRepoSourceId source id: %s, url: %s, source: %s.error:  %+v", j.SourceID, j.URL, j.RepositorySource, err))
		}
	}

	if j.RepositorySource == "gerrit" {
		sourceID, err = j.getGerritRepoSourceId()
		if err != nil {
			j.log.WithFields(logrus.Fields{"operation": "sync"}).Error(fmt.Errorf("getGerritRepoSourceId source id: %s, url: %s, source: %s.error:  %+v", j.SourceID, j.URL, j.RepositorySource, err))
		}
	}

	if sourceID != "" {
		j.SourceID = sourceID
	}
	// Continue with operations that need git ops
	nThreads := 0
	locFinished := false
	waitForLOC := func() (e error) {
		if thrN == 1 {
			locFinished = true
			return
		}
		waitLOCMtx.Lock()
		if !locFinished {
			if ctx.Debug > 0 {
				j.log.WithFields(logrus.Fields{"operation": "Sync"}).Debug("waiting for git ops result")
			}
			e1 := <-goch
			e2 := <-occh
			if e1 != nil && e2 != nil {
				e = fmt.Errorf("gitops error: %+v, orphaned commits error: %+v", e1, e2)
			} else {
				if e1 != nil {
					e = e1
				}
				if e2 != nil {
					e = e1
				}
			}
			if e != nil {
				waitLOCMtx.Unlock()
				return
			}
			locFinished = true
			if ctx.Debug > 0 {
				j.log.WithFields(logrus.Fields{"operation": "Sync"}).Debugf("loc: %d, programming languages: %d", j.Loc, len(j.Pls))
			}
		}
		waitLOCMtx.Unlock()
		return
	}
	processCommit := func(c chan error, commit map[string]interface{}) (wch chan error, e error) {
		sha, _ := commit["commit"].(string)
		cmdLine := []string{"cloc", "commit", sha, "--json"}
		sout, serr, err := shared.ExecCommand(ctx, cmdLine, j.GitPath, GitDefaultEnv)
		if err != nil {
			j.log.WithFields(logrus.Fields{"operation": "Sync"}).Errorf("error executing command: %v, error: %v, output: %s, output error: %s", cmdLine, err, sout, serr)
			return
		}
		r := make(map[string]clocResult)
		err = jsoniter.Unmarshal([]byte(sout), &r)
		if err != nil {
			j.log.WithFields(logrus.Fields{"operation": "Sync"}).Errorf("error unmarshall: %v, error: %v", sout, err)
			return
		}
		commit["cloc_count"] = r["SUM"].Code
		defer func() {
			if c != nil {
				c <- e
			}
		}()
		esItem := j.AddMetadata(ctx, commit)
		if ctx.Project != "" {
			commit["project"] = ctx.Project
		}
		e = waitForLOC()
		if e != nil {
			return
		}
		commit["total_lines_of_code"] = j.Loc
		commit["program_language_summary"] = j.Pls
		esItem["data"] = commit
		if allCommitsMtx != nil {
			allCommitsMtx.Lock()
		}
		allCommits = append(allCommits, esItem)
		nCommits := len(allCommits)
		if nCommits >= ctx.PackSize {
			sendToQueue := func(c chan error) (ee error) {
				defer func() {
					if c != nil {
						c <- ee
					}
				}()
				// ee = SendToQueue(ctx, j, true, UUID, allCommits)
				ee = j.GitEnrichItems(ctx, thrN, allCommits, &allDocs, false)
				if ee != nil {
					j.log.WithFields(logrus.Fields{"operation": "Sync"}).Errorf("error %v sending %d commits to queue", ee, len(allCommits))
				}
				allCommits = []interface{}{}
				if allCommitsMtx != nil {
					allCommitsMtx.Unlock()
				}
				return
			}
			if thrN > 1 {
				wch = make(chan error)
				go func() {
					_ = sendToQueue(wch)
				}()
			} else {
				e = sendToQueue(nil)
				if e != nil {
					return
				}
			}
		} else {
			if allCommitsMtx != nil {
				allCommitsMtx.Unlock()
			}
		}
		return
	}
	var (
		commit map[string]interface{}
		ok     bool
	)
	if thrN > 1 {
		for {
			commit, ok, err = j.ParseNextCommit(ctx)
			if err != nil {
				return
			}
			if !ok {
				break
			}
			go func(com map[string]interface{}) {
				var (
					e    error
					esch chan error
				)
				esch, e = processCommit(ch, com)
				if e != nil {
					j.log.WithFields(logrus.Fields{"operation": "Sync"}).Errorf("process error: %v", e)
					return
				}
				if esch != nil {
					if eschaMtx != nil {
						eschaMtx.Lock()
					}
					escha = append(escha, esch)
					if eschaMtx != nil {
						eschaMtx.Unlock()
					}
				}
			}(commit)
			nThreads++
			if nThreads == thrN {
				err = <-ch
				if err != nil {
					return
				}
				nThreads--
			}
		}
		for nThreads > 0 {
			err = <-ch
			nThreads--
			if err != nil {
				return
			}
		}
	} else {
		for {
			commit, ok, err = j.ParseNextCommit(ctx)
			if err != nil {
				return
			}
			if !ok {
				break
			}
			_, err = processCommit(nil, commit)
			if err != nil {
				return
			}
		}
	}
	// NOTE: lock needed
	if eschaMtx != nil {
		eschaMtx.Lock()
	}
	for _, esch := range escha {
		err = <-esch
		if err != nil {
			if eschaMtx != nil {
				eschaMtx.Unlock()
			}
			return
		}
	}
	if eschaMtx != nil {
		eschaMtx.Unlock()
	}
	err = cmd.Wait()
	if err != nil {
		return
	}
	nCommits := len(allCommits)
	if ctx.Debug > 0 {
		j.log.WithFields(logrus.Fields{"operation": "Sync"}).Debugf("%d remaining commits to send to queue", nCommits)
	}
	// NOTE: for all items, even if 0 - to flush the queue
	// err = SendToQueue(ctx, j, true, UUID, allCommits)
	err = j.GitEnrichItems(ctx, thrN, allCommits, &allDocs, true)
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "Sync"}).Errorf("Error %v sending %d commits to queue", err, len(allCommits))
	}
	if !locFinished {
		go func() {
			if ctx.Debug > 0 {
				j.log.WithFields(logrus.Fields{"operation": "Sync"}).Debug("gitops and orphaned commits result not needed, but waiting for orphan process")
			}
			<-goch
			<-occh
			locFinished = true
			if ctx.Debug > 0 {
				j.log.WithFields(logrus.Fields{"operation": "Sync"}).Debug("gitops and orphaned commits result not needed, but waiting for orphan process")
			}
		}()
	}

	if lastSync != "" {
		j.handleDataLakeOrphans()
	}
	// NOTE: Non-generic ends here
	err = j.setLastSync(ctx)
	return
}

func (j *DSGit) SyncV2(ctx *shared.Ctx) (err error) {
	thrN := 1 //shared.GetThreadsNum(ctx)
	lastSync := os.Getenv("LAST_SYNC")
	if lastSync != "" {
		i, err := strconv.ParseInt(lastSync, 10, 64)
		if err != nil {
			return err
		}
		tm := time.Unix(i, 0).UTC()
		ctx.DateFrom = &tm
	}
	if ctx.DateFrom != nil {
		j.log.WithFields(logrus.Fields{"operation": "Sync"}).Infof("%s fetching from %v (%d threads)", j.URL, ctx.DateFrom, thrN)
	}
	if ctx.DateFrom == nil {
		lastSyncDataB, er := j.cacheProvider.GetLastSyncFile(j.endpoint)
		if er != nil {
			err = er
			return
		}
		var lastSyncData lastSyncFile
		if er = jsoniter.Unmarshal(lastSyncDataB, &lastSyncData); er != nil {
			var cachedLastSync time.Time
			err = jsoniter.Unmarshal(lastSyncDataB, &cachedLastSync)
			if err != nil {
				err = er
				return
			}
			lastSyncData = lastSyncFile{
				LastSync: cachedLastSync,
			}
		}
		ctx.DateFrom = &lastSyncData.LastSync
		if ctx.DateFrom != nil {
			j.log.WithFields(logrus.Fields{"operation": "Sync"}).Infof("%s resuming from %v (%d threads)", j.URL, ctx.DateFrom, thrN)
		}
	}
	if ctx.DateTo != nil {
		j.log.WithFields(logrus.Fields{"operation": "Sync"}).Infof("%s fetching till %v (%d threads)", j.URL, ctx.DateTo, thrN)
	}
	// NOTE: Non-generic starts here
	var (
		ch            chan error
		allDocs       []interface{}
		allCommits    []interface{}
		allCommitsMtx *sync.Mutex
		escha         []chan error
		eschaMtx      *sync.Mutex
		goch          chan error
		occh          chan error
		waitLOCMtx    *sync.Mutex
	)
	if thrN > 1 {
		ch = make(chan error)
		allCommitsMtx = &sync.Mutex{}
		eschaMtx = &sync.Mutex{}
		waitLOCMtx = &sync.Mutex{}
		goch, _ = j.GetGitOps(ctx, thrN)
	} else {
		_, err = j.GetGitOps(ctx, thrN)
		if err != nil {
			return
		}
	}
	// Do normal git processing, which don't needs gitops yet
	j.GitPath = j.ReposPath + "/" + j.URL + "-git"
	j.GitPath, err = shared.EnsurePath(j.GitPath, true)
	shared.FatalOnError(err)
	if ctx.Debug > 0 {
		j.log.WithFields(logrus.Fields{"operation": "Sync"}).Debugf("path to store git repository: %s", j.GitPath)
	}

	r, err := j.cloneRepoCommand(ctx)
	if err != nil {
		return err
	}

	firstCommit, err := j.getFirstCommit(ctx, r)
	if err != nil {
		return err
	}
	FirstCommitAt = firstCommit.Author.When
	from := firstCommit.Author.When.Add(time.Second * -60)
	if ctx.DateFrom.After(from) {
		from = *ctx.DateFrom
	}

	ref, err := r.Head()
	if err != nil {
		return err
	}

	j.headCommitHash = ref.Hash().String()
	headCommit, err := r.CommitObject(plumbing.NewHash(ref.Hash().String()))
	if err != nil {
		return err
	}
	if err = j.getCloc(ctx, j.headCommitHash); err != nil {
		return err
	}
	if thrN > 1 {
		occh, _ = j.GetOrphanedCommits(ctx, thrN)
	} else {
		_, err = j.GetOrphanedCommits(ctx, thrN)
		if err != nil {
			return
		}
	}
	err = j.GetGitBranches(ctx)
	if err != nil {
		return
	}

	sourceID := ""
	if j.RepositorySource == "github" {
		sourceID, err = j.getGithubRepoSourceId()
		if err != nil {
			j.log.WithFields(logrus.Fields{"operation": "sync"}).Error(fmt.Errorf("getGithubRepoSourceId source id: %s, url: %s, source: %s.error:  %+v", j.SourceID, j.URL, j.RepositorySource, err))
		}
	}

	if j.RepositorySource == "gerrit" {
		sourceID, err = j.getGerritRepoSourceId()
		if err != nil {
			j.log.WithFields(logrus.Fields{"operation": "sync"}).Error(fmt.Errorf("getGerritRepoSourceId source id: %s, url: %s, source: %s.error:  %+v", j.SourceID, j.URL, j.RepositorySource, err))
		}
	}

	if sourceID != "" {
		j.SourceID = sourceID
	}
	// Continue with operations that need git ops
	nThreads := 0
	locFinished := false
	waitForLOC := func() (e error) {
		if thrN == 1 {
			locFinished = true
			return
		}
		waitLOCMtx.Lock()
		if !locFinished {
			if ctx.Debug > 0 {
				j.log.WithFields(logrus.Fields{"operation": "Sync"}).Debug("waiting for git ops result")
			}
			e1 := <-goch
			e2 := <-occh
			if e1 != nil && e2 != nil {
				e = fmt.Errorf("gitops error: %+v, orphaned commits error: %+v", e1, e2)
			} else {
				if e1 != nil {
					e = e1
				}
				if e2 != nil {
					e = e1
				}
			}
			if e != nil {
				waitLOCMtx.Unlock()
				return
			}
			locFinished = true
			if ctx.Debug > 0 {
				j.log.WithFields(logrus.Fields{"operation": "Sync"}).Debugf("loc: %d, programming languages: %d", j.Loc, len(j.Pls))
			}
		}
		waitLOCMtx.Unlock()
		return
	}
	processCommit := func(c chan error, commit map[string]interface{}) (wch chan error, e error) {
		sha, _ := commit["commit"].(string)
		if sha == j.headCommitHash {
			commit["cloc_count"] = j.headLinesOfCode
		}
		defer func() {
			if c != nil {
				c <- e
			}
		}()
		esItem := j.AddMetadata(ctx, commit)
		if ctx.Project != "" {
			commit["project"] = ctx.Project
		}
		e = waitForLOC()
		if e != nil {
			return
		}
		commit["total_lines_of_code"] = j.Loc
		commit["program_language_summary"] = j.Pls
		esItem["data"] = commit
		if allCommitsMtx != nil {
			allCommitsMtx.Lock()
		}
		allCommits = append(allCommits, esItem)
		nCommits := len(allCommits)
		if nCommits >= ctx.PackSize {
			sendToQueue := func(c chan error) (ee error) {
				defer func() {
					if c != nil {
						c <- ee
					}
				}()
				// ee = SendToQueue(ctx, j, true, UUID, allCommits)
				ee = j.GitEnrichItems(ctx, thrN, allCommits, &allDocs, false)
				if ee != nil {
					j.log.WithFields(logrus.Fields{"operation": "Sync"}).Errorf("error %v sending %d commits to queue", ee, len(allCommits))
				}
				allCommits = []interface{}{}
				if allCommitsMtx != nil {
					allCommitsMtx.Unlock()
				}
				return
			}
			if thrN > 1 {
				wch = make(chan error)
				go func() {
					_ = sendToQueue(wch)
				}()
			} else {
				e = sendToQueue(nil)
				if e != nil {
					return
				}
			}
		} else {
			if allCommitsMtx != nil {
				allCommitsMtx.Unlock()
			}
		}
		return
	}

	commitsCount, err := j.getCommitsCount(ctx)
	if err != nil {
		return err
	}
	if commitsCount >= HotRepoCount {
		IsHotRep = true
		CurrentCacheYear = from.Year()
		if int(from.Month()) > 6 {
			CurrentCacheYearHalf = YearSecondHalf
		}
		j.getYearHalfCache(lastSync)
		j.getUpdateCache(lastSync)
	} else {
		j.getCache(lastSync)
	}

	for from.Before(headCommit.Author.When) {
		until := from.Add(24 * time.Hour * 30)
		comms, er := getRepoCommits(r, from, until)
		if er != nil {
			err = er
			return
		}
		if thrN > 1 {
			for i := len(comms) - 1; i >= 0; i-- {
				var c map[string]interface{}
				c, err = j.BuildCommitMap(comms[i])
				if err != nil {
					return err
				}
				var (
					e    error
					esch chan error
				)
				esch, e = processCommit(ch, c)
				if e != nil {
					j.log.WithFields(logrus.Fields{"operation": "Sync"}).Errorf("process error: %v", e)
					return
				}
				if esch != nil {
					if eschaMtx != nil {
						eschaMtx.Lock()
					}
					escha = append(escha, esch)
					if eschaMtx != nil {
						eschaMtx.Unlock()
					}
				}
				nThreads++
				if nThreads == thrN {
					err = <-ch
					if err != nil {
						return
					}
					nThreads--
				}
			}
			for nThreads > 0 {
				err = <-ch
				nThreads--
				if err != nil {
					return
				}
			}
		} else {

			for i := len(comms) - 1; i >= 0; i-- {
				var com map[string]interface{}
				com, err = j.BuildCommitMap(comms[i])
				if err != nil {
					return err
				}
				_, err = processCommit(nil, com)
				if err != nil {
					return
				}
			}

		}
		from = until
		until = until.Add(24 * time.Hour)
	}
	// NOTE: lock needed
	if eschaMtx != nil {
		eschaMtx.Lock()
	}
	for _, esch := range escha {
		err = <-esch
		if err != nil {
			if eschaMtx != nil {
				eschaMtx.Unlock()
			}
			return
		}
	}
	if eschaMtx != nil {
		eschaMtx.Unlock()
	}
	if err != nil {
		return
	}
	nCommits := len(allCommits)
	if ctx.Debug > 0 {
		j.log.WithFields(logrus.Fields{"operation": "Sync"}).Debugf("%d remaining commits to send to queue", nCommits)
	}
	// NOTE: for all items, even if 0 - to flush the queue
	// err = SendToQueue(ctx, j, true, UUID, allCommits)
	err = j.GitEnrichItems(ctx, thrN, allCommits, &allDocs, true)
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "Sync"}).Errorf("Error %v sending %d commits to queue", err, len(allCommits))
	}
	if !locFinished {
		go func() {
			if ctx.Debug > 0 {
				j.log.WithFields(logrus.Fields{"operation": "Sync"}).Debug("gitops and orphaned commits result not needed, but waiting for orphan process")
			}
			<-goch
			<-occh
			locFinished = true
			if ctx.Debug > 0 {
				j.log.WithFields(logrus.Fields{"operation": "Sync"}).Debug("gitops and orphaned commits result not needed, but waiting for orphan process")
			}
		}()
	}

	if lastSync != "" {
		j.handleDataLakeOrphans()
	}
	// NOTE: Non-generic ends here
	err = j.setLastSync(ctx)
	return
}

func main() {
	var (
		ctx shared.Ctx
		git DSGit
	)
	git.initStructuredLogger()
	err := git.Init(&ctx)
	if err != nil {
		git.log.WithFields(logrus.Fields{"operation": "main"}).Errorf("Error: %+v", err)
		return
	}
	git.log = git.log.WithFields(
		logrus.Fields{
			"endpoint": git.URL,
		})
	timestamp := time.Now()
	shared.SetSyncMode(true, false)
	shared.SetLogLoggerError(false)
	shared.AddLogger(&git.Logger, GitDataSource, logger.Internal, []map[string]string{{"REPO_URL": git.URL, "ProjectSlug": ctx.Project}})
	git.AddCacheProvider()
	git.AddReportProvider()
	/*	if os.Getenv("SPAN") != "" {
		tracer.Start(tracer.WithGlobalTag("connector", "git"))
		defer tracer.Stop()

		sb := os.Getenv("SPAN")
		carrier := make(tracer.TextMapCarrier)
		err = jsoniter.Unmarshal([]byte(sb), &carrier)
		if err != nil {
			return
		}
		sctx, er := tracer.Extract(carrier)
		if er != nil {
			fmt.Println(er)
		}
		if err == nil && sctx != nil {
			span, _ := tracer.StartSpanFromContext(context.Background(), "commit", tracer.ResourceName("connector"), tracer.ChildOf(sctx))
			defer span.Finish()
		}
	}*/

	err = git.WriteLog(&ctx, timestamp, logger.InProgress, "")
	if err != nil {
		git.log.WithFields(logrus.Fields{"operation": "main"}).Errorf("WriteLog Error : %+v", err)
	}
	shared.FatalOnError(err)
	if err = git.addAuth0Client(); err != nil {
		git.log.WithFields(logrus.Fields{"operation": "main"}).Errorf("addAuth0Client Error : %+v", err)
	}
	err = git.SyncV2(&ctx)
	if err != nil {
		git.log.WithFields(logrus.Fields{"operation": "main"}).Errorf("Error: %+v", err)
		er := git.WriteLog(&ctx, timestamp, logger.Failed, err.Error())
		if er != nil {
			git.log.WithFields(logrus.Fields{"operation": "main"}).Errorf("WriteLog Error : %+v", err)
			shared.FatalOnError(er)
		}
	}
	shared.FatalOnError(err)

	err = git.WriteLog(&ctx, timestamp, logger.Done, "")
	if err != nil {
		git.log.WithFields(logrus.Fields{"operation": "main"}).Errorf("WriteLog Error : %+v", err)
	}
	shared.FatalOnError(err)
}

// createStructuredLogger...
func (j *DSGit) initStructuredLogger() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	log := logrus.WithFields(
		logrus.Fields{
			"environment": os.Getenv("STAGE"),
			"commit":      build.GitCommit,
			"version":     build.Version,
			"service":     build.AppName,
			"endpoint":    j.URL,
		})
	j.log = log
}

// AddCacheProvider - adds cache provider
func (j *DSGit) AddCacheProvider() {
	cacheProvider := cache.NewManager(fmt.Sprintf("v2/%s", GitDataSource), os.Getenv("STAGE"))
	j.cacheProvider = *cacheProvider
	j.endpoint = strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(j.URL, "https://"), "git://"), "http://"), "/", "-")
}

// AddReportProvider - adds report provider
func (j *DSGit) AddReportProvider() {
	reportProvider := report.NewManager(os.Getenv("STAGE"))
	j.reportProvider = reportProvider
	j.endpoint = strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(j.URL, "https://"), "git://"), "http://"), "/", "-")
}

func (j *DSGit) createCacheFile(cache []CommitCache, path string) error {
	for _, comm := range cache {
		comm.FileLocation = path
		cachedCommits[comm.EntityID] = comm
	}
	records := [][]string{
		{"timestamp", "entity_id", "source_entity_id", "file_location", "hash", "orphaned", "from_dl", "content"},
	}
	for _, c := range cachedCommits {
		records = append(records, []string{c.Timestamp, c.EntityID, c.SourceEntityID, c.FileLocation, c.Hash, strconv.FormatBool(c.Orphaned), strconv.FormatBool(c.FromDL), c.Content})
	}

	csvFile, err := os.Create(commitsCacheFile)
	if err != nil {
		return err
	}

	w := csv.NewWriter(csvFile)
	err = w.WriteAll(records)
	if err != nil {
		return err
	}
	err = csvFile.Close()
	if err != nil {
		return err
	}
	file, err := os.ReadFile(commitsCacheFile)
	if err != nil {
		return err
	}
	err = os.Remove(commitsCacheFile)
	if err != nil {
		return err
	}
	err = j.cacheProvider.UpdateFileByKey(j.endpoint, commitsCacheFile, file)
	if err != nil {
		return err
	}

	return nil
}

func (j *DSGit) createYearHalfCacheFileOld(cache []CommitCache, path string) error {
	nextYearHalfCache := make([]CommitCache, 0)
	currentYearCommitsCount := 0
	for _, comm := range cache {
		comm.FileLocation = path
		commitYearHalf := getDateYearHalf(comm.CommitDate)
		if comm.CommitDate.Year() == CurrentCacheYear && commitYearHalf == CurrentCacheYearHalf {
			cachedCommits[comm.EntityID] = comm
			currentYearCommitsCount++
		} else {
			nextYearHalfCache = append(nextYearHalfCache, comm)
		}
	}
	if currentYearCommitsCount == 0 {
		return nil
	}
	records := [][]string{
		{"timestamp", "entity_id", "source_entity_id", "file_location", "hash", "orphaned", "from_dl", "content"},
	}
	for _, c := range cachedCommits {
		records = append(records, []string{c.Timestamp, c.EntityID, c.SourceEntityID, c.FileLocation, c.Hash, strconv.FormatBool(c.Orphaned), strconv.FormatBool(c.FromDL), c.Content})
	}

	yearSTR := strconv.Itoa(CurrentCacheYear)
	cacheFile := fmt.Sprintf(CommitsByYearHalfCacheFile, yearSTR, CurrentCacheYearHalf)
	csvFile, err := os.Create(cacheFile)
	if err != nil {
		return err
	}

	w := csv.NewWriter(csvFile)
	err = w.WriteAll(records)
	if err != nil {
		return err
	}
	err = csvFile.Close()
	if err != nil {
		return err
	}
	err = j.cacheProvider.UpdateMultiPartFileByKey(j.endpoint, cacheFile)
	if err != nil {
		return err
	}

	err = os.Remove(cacheFile)
	if err != nil {
		return err
	}

	if len(nextYearHalfCache) > 0 {
		CurrentCacheYear = nextYearHalfCache[0].CommitDate.Year()
		CurrentCacheYearHalf = YearFirstHalf
		if nextYearHalfCache[0].CommitDate.Month() > 6 {
			CurrentCacheYearHalf = YearSecondHalf
		}
		//updateYearHalf(nextYearHalfCache[0].CommitDate)
		if err = j.createYearHalfCacheFile(nextYearHalfCache, path); err != nil {
			return err
		}
		cachedCommits = make(map[string]CommitCache)
		j.getYearHalfCache(os.Getenv("LAST_SYNC"))
	}
	return nil
}

func (j *DSGit) createYearHalfCacheFile(cache []CommitCache, path string) error {
	nextYearHalfCache := make([]CommitCache, 0)
	for _, comm := range cache {
		comm.FileLocation = path
		commitYearHalf := getDateYearHalf(comm.CommitDate)
		if comm.CommitDate.Year() == CurrentCacheYear && commitYearHalf == CurrentCacheYearHalf {
			cachedCommits[comm.EntityID] = comm
		} else {
			nextYearHalfCache = append(nextYearHalfCache, comm)
		}
	}

	if err := j.syncRemoteCurrentYearCache(); err != nil {
		return err
	}

	if len(nextYearHalfCache) > 0 {
		CurrentCacheYear = nextYearHalfCache[0].CommitDate.Year()
		CurrentCacheYearHalf = YearFirstHalf
		if nextYearHalfCache[0].CommitDate.Month() > 6 {
			CurrentCacheYearHalf = YearSecondHalf
		}

		j.getYearHalfCache(os.Getenv("LAST_SYNC"))
		for _, comm := range nextYearHalfCache {
			comm.FileLocation = path
			cachedCommits[comm.EntityID] = comm
		}

		if err := j.syncRemoteCurrentYearCache(); err != nil {
			return err
		}
	}

	return nil
}

func (j *DSGit) syncRemoteCurrentYearCache() error {
	records := [][]string{
		{"timestamp", "entity_id", "source_entity_id", "file_location", "hash", "orphaned", "from_dl", "content"},
	}
	for _, c := range cachedCommits {
		records = append(records, []string{c.Timestamp, c.EntityID, c.SourceEntityID, c.FileLocation, c.Hash, strconv.FormatBool(c.Orphaned), strconv.FormatBool(c.FromDL), c.Content})
	}

	yearSTR := strconv.Itoa(CurrentCacheYear)
	cacheFile := fmt.Sprintf(CommitsByYearHalfCacheFile, yearSTR, CurrentCacheYearHalf)
	csvFile, err := os.Create(cacheFile)
	if err != nil {
		return err
	}

	w := csv.NewWriter(csvFile)
	err = w.WriteAll(records)
	if err != nil {
		return err
	}
	err = csvFile.Close()
	if err != nil {
		return err
	}
	err = j.cacheProvider.UpdateMultiPartFileByKey(j.endpoint, cacheFile)
	if err != nil {
		return err
	}

	err = os.Remove(cacheFile)
	if err != nil {
		return err
	}

	return nil
}

func getDateYearHalf(commitDate time.Time) string {
	monthNumber := int(commitDate.Month())
	if monthNumber > 6 {
		return YearSecondHalf
	}
	return YearFirstHalf
}

func updateYearHalf(commitDate time.Time) {
	cuHalf := getDateYearHalf(commitDate)
	if cuHalf == CurrentCacheYearHalf {
		return
	}

	if CurrentCacheYearHalf == YearFirstHalf {
		CurrentCacheYearHalf = YearSecondHalf
		return
	}
	CurrentCacheYearHalf = YearFirstHalf
	CurrentCacheYear += 1
}

func (j *DSGit) createUpdateCacheFile(cache []CommitCache, path string) error {
	for _, comm := range cache {
		comm.FileLocation = path
		CachedCommitsUpdates[comm.EntityID] = comm
	}
	records := [][]string{
		{"timestamp", "entity_id", "source_entity_id", "file_location", "hash", "orphaned", "from_dl", "content"},
	}
	for _, c := range CachedCommitsUpdates {
		records = append(records, []string{c.Timestamp, c.EntityID, c.SourceEntityID, c.FileLocation, c.Hash, strconv.FormatBool(c.Orphaned), strconv.FormatBool(c.FromDL), c.Content})
	}

	csvFile, err := os.Create(CommitsUpdateCacheFile)
	if err != nil {
		return err
	}

	w := csv.NewWriter(csvFile)
	err = w.WriteAll(records)
	if err != nil {
		return err
	}
	err = csvFile.Close()
	if err != nil {
		return err
	}
	file, err := os.ReadFile(CommitsUpdateCacheFile)
	if err != nil {
		return err
	}
	err = os.Remove(CommitsUpdateCacheFile)
	if err != nil {
		return err
	}
	err = j.cacheProvider.UpdateFileByKey(j.endpoint, CommitsUpdateCacheFile, file)
	if err != nil {
		return err
	}

	return nil
}

func isHashCreated(hash string) bool {
	c, ok := cachedCommits[hash]
	if ok {
		c.Orphaned = false
		cachedCommits[hash] = c
		return true
	}
	return false
}

func (j *DSGit) getCache(lastSync string) {
	commentBytes, err := j.cacheProvider.GetFileByKey(j.endpoint, commitsCacheFile)
	if err != nil {
		return
	}
	reader := csv.NewReader(bytes.NewBuffer(commentBytes))
	records, err := reader.ReadAll()
	if err != nil {
		return
	}
	for i, record := range records {
		if i == 0 {
			continue
		}
		orphaned, err := strconv.ParseBool(record[5])
		if err != nil {
			orphaned = false
		}
		if lastSync != "" {
			orphaned = true
		}

		var fromDL bool
		if len(record) > 6 {
			fromDL, err = strconv.ParseBool(record[6])
			if err != nil {
				fromDL = false
			}
		}

		var content string
		if len(record) > 7 {
			if record[7] != "" {
				content = record[7]
			}
		}

		cachedCommits[record[4]] = CommitCache{
			Timestamp:      record[0],
			EntityID:       record[1],
			SourceEntityID: record[2],
			FileLocation:   record[3],
			Hash:           record[4],
			Orphaned:       orphaned,
			FromDL:         fromDL,
			Content:        content,
		}

		createdCommits[record[1]] = true
	}
}

func (j *DSGit) getUpdateCache(lastSync string) {
	commentBytes, err := j.cacheProvider.GetFileByKey(j.endpoint, CommitsUpdateCacheFile)
	if err != nil {
		return
	}
	reader := csv.NewReader(bytes.NewBuffer(commentBytes))
	records, err := reader.ReadAll()
	if err != nil {
		return
	}
	for i, record := range records {
		if i == 0 {
			continue
		}
		orphaned, err := strconv.ParseBool(record[5])
		if err != nil {
			orphaned = false
		}
		if lastSync != "" {
			orphaned = true
		}

		var fromDL bool
		if len(record) > 6 {
			fromDL, err = strconv.ParseBool(record[6])
			if err != nil {
				fromDL = false
			}
		}

		var content string
		if len(record) > 7 {
			if record[7] != "" {
				content = record[7]
			}
		}

		CachedCommitsUpdates[record[4]] = CommitCache{
			Timestamp:      record[0],
			EntityID:       record[1],
			SourceEntityID: record[2],
			FileLocation:   record[3],
			Hash:           record[4],
			Orphaned:       orphaned,
			FromDL:         fromDL,
			Content:        content,
		}

		createdCommits[record[1]] = true
	}
}

func (j *DSGit) getYearHalfCache(lastSync string) {
	yearSTR := strconv.Itoa(CurrentCacheYear)
	commentBytes, err := j.cacheProvider.GetFileByKey(j.endpoint, fmt.Sprintf(CommitsByYearHalfCacheFile, yearSTR, CurrentCacheYearHalf))
	if err != nil {
		return
	}
	reader := csv.NewReader(bytes.NewBuffer(commentBytes))
	records, err := reader.ReadAll()
	if err != nil {
		return
	}
	for i, record := range records {
		if i == 0 {
			continue
		}
		orphaned, err := strconv.ParseBool(record[5])
		if err != nil {
			orphaned = false
		}
		if lastSync != "" {
			orphaned = true
		}

		var fromDL bool
		if len(record) > 6 {
			fromDL, err = strconv.ParseBool(record[6])
			if err != nil {
				fromDL = false
			}
		}

		var content string
		if len(record) > 7 {
			if record[7] != "" {
				content = record[7]
			}
		}

		cachedCommits[record[4]] = CommitCache{
			Timestamp:      record[0],
			EntityID:       record[1],
			SourceEntityID: record[2],
			FileLocation:   record[3],
			Hash:           record[4],
			Orphaned:       orphaned,
			FromDL:         fromDL,
			Content:        content,
		}

		createdCommits[record[1]] = true
	}
}

func (j *DSGit) getCacheFileByKey(key string, lastSync string) (map[string]CommitCache, error) {
	commits := make(map[string]CommitCache)
	commentBytes, err := j.cacheProvider.GetFileByKey(j.endpoint, key)
	if err != nil {
		return commits, err
	}
	reader := csv.NewReader(bytes.NewBuffer(commentBytes))
	records, err := reader.ReadAll()
	if err != nil {
		return commits, err
	}
	for i, record := range records {
		if i == 0 {
			continue
		}
		orphaned, err := strconv.ParseBool(record[5])
		if err != nil {
			orphaned = false
		}
		if lastSync != "" {
			orphaned = true
		}

		var fromDL bool
		if len(record) > 6 {
			fromDL, err = strconv.ParseBool(record[6])
			if err != nil {
				fromDL = false
			}
		}

		var content string
		if len(record) > 7 {
			if record[7] != "" {
				content = record[7]
			}
		}

		commits[record[4]] = CommitCache{
			Timestamp:      record[0],
			EntityID:       record[1],
			SourceEntityID: record[2],
			FileLocation:   record[3],
			Hash:           record[4],
			Orphaned:       orphaned,
			FromDL:         fromDL,
			Content:        content,
		}
	}
	return commits, nil
}

func isCommitCreated(id string) bool {
	_, ok := cachedCommits[id]
	return ok
}

// handleDataLakeOrphans Update commits in DL with new orphaned status
func (j *DSGit) handleDataLakeOrphans() {
	formattedData := make([]interface{}, 0)
	baseEvent := service.BaseEvent{
		Type: CommitUpdated,
		CRUDInfo: service.CRUDInfo{
			CreatedBy: GitConnector,
			UpdatedBy: GitConnector,
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		},
	}
	commitBaseEvent := git.CommitBaseEvent{
		Connector:        insights.GitConnector,
		ConnectorVersion: GitBackendVersion,
		Source:           insights.Source(j.RepositorySource),
	}

	for _, v := range cachedCommits {
		if v.Orphaned {
			commitB, err := b64.StdEncoding.DecodeString(v.Content)
			if err != nil {
				j.log.WithFields(logrus.Fields{"operation": "handleDataLakeOrphans"}).Errorf("error decode datalake orphand commit: %+v", err)
			}
			var commit git.Commit
			err = jsoniter.Unmarshal(commitB, &commit)
			if err != nil {
				j.log.WithFields(logrus.Fields{"operation": "handleDataLakeOrphans"}).Errorf("error unmarshall datalake orphand commit: %+v", err)
				continue
			}
			commit.Orphaned = true
			commitEvent := git.CommitUpdatedEvent{
				CommitBaseEvent: commitBaseEvent,
				BaseEvent:       baseEvent,
				Payload:         commit,
			}
			formattedData = append(formattedData, commitEvent)
		}
	}

	if len(formattedData) > 0 {
		path, err := j.Publisher.PushEvents(CommitUpdated, "insights", GitDataSource, "commits", os.Getenv("STAGE"), formattedData, j.endpoint)
		if err != nil {
			j.log.WithFields(logrus.Fields{"operation": "handleDataLakeOrphans"}).Errorf("error pushing data lake orphand commits: %+v", err)
			return
		}
		for _, c := range formattedData {
			payload := c.(git.CommitUpdatedEvent).Payload
			contentHash, er := createHash(payload)
			if er != nil {
				j.log.WithFields(logrus.Fields{"operation": "handleDataLakeOrphans"}).Errorf("error hashing commit data: %+v", err)
				continue
			}
			commit := cachedCommits[contentHash]
			commit.FromDL = false
			commit.FileLocation = path
			commit.Content = ""
			cachedCommits[contentHash] = commit
		}
		if err = j.createUpdateCacheFile([]CommitCache{}, ""); err != nil {
			j.log.WithFields(logrus.Fields{"operation": "handleDataLakeOrphans"}).Errorf("error updating commits cache: %+v", err)
			return
		}
		j.log.WithFields(logrus.Fields{"operation": "handleDataLakeOrphans"}).Infof("updated %d orphand commits from data lake", len(formattedData))
	}

}

// handleHotRepoDataLakeOrphans Update hot repository commits in DL with new orphaned status
func (j *DSGit) handleHotRepoDataLakeOrphans() {
	year := FirstCommitAt.Year()
	half := YearFirstHalf
	yearSTR := strconv.Itoa(year)

	cacheFileName := fmt.Sprintf(CommitsByYearHalfCacheFile, yearSTR, half)
	for {
		commits, err := j.getCacheFileByKey(cacheFileName, "")
		if err != nil {
			if year > CurrentCacheYear {
				break
			}
			if year == CurrentCacheYear && CurrentCacheYearHalf == YearFirstHalf && half == YearSecondHalf {
				break
			}
			continue
		}

		if half == YearSecondHalf {
			year++
			half = YearFirstHalf
		}
		if half == YearFirstHalf {
			half = YearSecondHalf
		}

		formattedData := j.handleSingleCacheFile(commits)
		if len(formattedData) > 0 {
			path, err := j.Publisher.PushEvents(CommitUpdated, "insights", GitDataSource, "commits", os.Getenv("STAGE"), formattedData, j.endpoint)
			if err != nil {
				j.log.WithFields(logrus.Fields{"operation": "handleDataLakeOrphans"}).Errorf("error pushing data lake orphand commits: %+v", err)
				return
			}
			for _, c := range formattedData {
				payload := c.(git.CommitUpdatedEvent).Payload
				contentHash, er := createHash(payload)
				if er != nil {
					j.log.WithFields(logrus.Fields{"operation": "handleDataLakeOrphans"}).Errorf("error hashing commit data: %+v", err)
					continue
				}
				commit := cachedCommits[contentHash]
				commit.FromDL = false
				commit.FileLocation = path
				commit.Content = ""
				cachedCommits[contentHash] = commit
			}
			if err = j.createUpdateCacheFile([]CommitCache{}, ""); err != nil {
				j.log.WithFields(logrus.Fields{"operation": "handleDataLakeOrphans"}).Errorf("error updating commits cache: %+v", err)
				return
			}
			j.log.WithFields(logrus.Fields{"operation": "handleDataLakeOrphans"}).Infof("updated %d orphand commits from data lake", len(formattedData))
		}
	}

}

func (j *DSGit) handleSingleCacheFile(commits map[string]CommitCache) []interface{} {
	formattedData := make([]interface{}, 0)
	baseEvent := service.BaseEvent{
		Type: CommitUpdated,
		CRUDInfo: service.CRUDInfo{
			CreatedBy: GitConnector,
			UpdatedBy: GitConnector,
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		},
	}
	commitBaseEvent := git.CommitBaseEvent{
		Connector:        insights.GitConnector,
		ConnectorVersion: GitBackendVersion,
		Source:           insights.Source(j.RepositorySource),
	}
	for _, v := range commits {
		if v.Orphaned {
			commitB, err := b64.StdEncoding.DecodeString(v.Content)
			if err != nil {
				j.log.WithFields(logrus.Fields{"operation": "handleDataLakeOrphans"}).Errorf("error decode datalake orphand commit: %+v", err)
			}
			var commit git.Commit
			err = jsoniter.Unmarshal(commitB, &commit)
			if err != nil {
				j.log.WithFields(logrus.Fields{"operation": "handleDataLakeOrphans"}).Errorf("error unmarshall datalake orphand commit: %+v", err)
				continue
			}
			commit.Orphaned = true
			commitEvent := git.CommitUpdatedEvent{
				CommitBaseEvent: commitBaseEvent,
				BaseEvent:       baseEvent,
				Payload:         commit,
			}
			formattedData = append(formattedData, commitEvent)
		}
	}
	return formattedData
}

func createHash(content git.Commit) (string, error) {
	commitHashFields := CommitHashFields{
		ID:            content.ID,
		SHA:           content.SHA,
		RepositoryURL: content.RepositoryURL,
		RepositoryID:  content.RepositoryID,
		Message:       content.Message,
	}
	b, err := jsoniter.Marshal(commitHashFields)
	if err != nil {
		return "", err
	}
	contentHash := fmt.Sprintf("%x", sha256.Sum256(b))

	return contentHash, err
}

func (j *DSGit) setLastSync(ctx *shared.Ctx) error {
	commitsCount, err := j.getCommitsCount(ctx)
	if err != nil {
		return err
	}

	commitID, err := j.getHead(ctx)
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "getHead"}).Warningf("error getting repository head %v", err)
	}

	gMaxUpstreamDtMtx.Lock()
	defer gMaxUpstreamDtMtx.Unlock()

	lastSyncData := lastSyncFile{
		LastSync:      gMaxUpstreamDt,
		Target:        commitsCount,
		Total:         len(createdCommits),
		Head:          commitID,
		FirstCommitAt: FirstCommitAt,
	}

	lastSyncDataB, err := jsoniter.Marshal(lastSyncData)
	if err != nil {
		return err
	}

	if !gMaxUpstreamDt.IsZero() {
		err = j.cacheProvider.SetLastSyncFile(j.endpoint, lastSyncDataB)
		if err != nil {
			return err
		}
	}

	return nil
}

func (j *DSGit) getCommitsCount(ctx *shared.Ctx) (int, error) {
	count := 0
	cmdLine := []string{"git", "rev-list", "--count", j.DefaultBranch}
	sout, serr, err := shared.ExecCommand(ctx, cmdLine, j.GitPath, GitDefaultEnv)
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "gitCommitsCount"}).Errorf("error executing command: %v, error: %v, output: %s, output error: %s", cmdLine, err, sout, serr)
		return count, err
	}
	result := strings.TrimSpace(sout)
	count, err = strconv.Atoi(result)
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "gitCommitsCount"}).Errorf("error converting: %v, to int error: %v", result, err)
		return count, err
	}
	return count, nil
}

func (j *DSGit) getHead(ctx *shared.Ctx) (string, error) {
	return j.headCommitHash, nil
}

func (j *DSGit) addAuth0Client() error {
	esCacheClientProvider, err := elastic.NewClientProvider(&elastic.Params{
		URL: os.Getenv("ES_CACHE_URL"),
	})
	if err != nil {
		return err
	}
	httpClientProvider := http.NewClientProvider(time.Second*50, true)
	authEnv := os.Getenv("STAGE")
	if authEnv == "dev" {
		authEnv = "test"
	}
	auth0Client, err := auth0.NewAuth0Client(
		authEnv,
		os.Getenv("AUTH_GRANT_TYPE"),
		os.Getenv("AUTH_CLIENT_ID"),
		os.Getenv("AUTH_CLIENT_SECRET"),
		os.Getenv("AUTH_AUDIENCE"),
		os.Getenv("AUTH0_URL"),
		httpClientProvider,
		esCacheClientProvider,
		nil,
		build.AppName)
	if err != nil {
		return err
	}
	j.auth0Client = auth0Client
	return nil
}

func (j *DSGit) getGithubRepoSourceId() (string, error) {
	sourceID := ""
	if j.auth0Client == nil {
		return "", nil
	}

	type response struct {
		SourceID int64 `json:"sourceId"`
	}

	httpClientProvider := http.NewClientProvider(time.Second*50, true)
	token, err := j.auth0Client.GetToken()
	if err != nil {
		return sourceID, err
	}
	headers := make(map[string]string)
	headers["Authorization"] = fmt.Sprintf("Bearer %s", token)
	orgRepoList := strings.Split(strings.TrimPrefix(j.URL, "https://github.com/"), "/")

	URL := fmt.Sprintf("%sproject/source/github/organization/%s/repository/%s", os.Getenv("INSIGHTS_SERVICE_URL_V2"), orgRepoList[0], orgRepoList[1])
	statusCode, res, err := httpClientProvider.Request(URL, "GET", headers, nil, nil)
	if err != nil {
		return sourceID, err
	}
	if statusCode != 200 {
		return sourceID, fmt.Errorf("error getting source id status code: %d", statusCode)
	}
	var repo response
	if err = jsoniter.Unmarshal(res, &repo); err != nil {
		return sourceID, err
	}
	if repo.SourceID != 0 {
		sourceID = strconv.FormatInt(repo.SourceID, 10)
	}
	return sourceID, nil
}

func (j *DSGit) getGerritRepoSourceId() (string, error) {
	sourceID := ""
	const (
		repoSeparatorGerrit = "/gerrit/"
		repoSeparatorR      = "/r/"
	)
	repoSeparatorList := []string{repoSeparatorR, repoSeparatorGerrit}
	for _, repoSeparator := range repoSeparatorList {
		repoSlice := strings.Split(j.URL, repoSeparator)
		if len(repoSlice) > 1 {
			if repoSeparator == repoSeparatorGerrit {
				nRepoSlice := strings.Split(j.URL, repoSeparatorR)
				if len(nRepoSlice) > 1 {
					sourceID = strings.TrimSpace(repoSlice[1])
					break
				}
			}
			sourceID = strings.TrimSpace(repoSlice[1])
			break
		}
	}

	// check if url already contains a partial
	hasPartial := false
	u, err := url.Parse(j.URL)
	if err != nil {
		return sourceID, err
	}

	for _, p := range repoSeparatorList {
		repoSlice := strings.Split(u.Path, p)
		if len(repoSlice) > 1 {
			hasPartial = true
			break
		}
	}

	if !hasPartial {
		repoSlice := strings.Split(j.URL, strings.TrimSpace(fmt.Sprintf("%s://%s", u.Scheme, u.Host)))
		if len(repoSlice) > 1 {
			sourceID = strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(repoSlice[1]), "/"), "/")
		}
	}
	return sourceID, nil
}

func (j *DSGit) cloneRepo() (*goGit.Repository, error) {
	r, err := goGit.PlainClone(j.GitPath, false, &goGit.CloneOptions{
		URL:      j.URL,
		Tags:     goGit.NoTags,
		Progress: os.Stdout,
	})
	if err != nil {
		return r, err
	}
	fmt.Println("repository cloned successfully")
	return r, nil
}

func (j *DSGit) cloneRepoCommand(ctx *shared.Ctx) (*goGit.Repository, error) {
	if err := j.CreateGitRepo(ctx); err != nil {
		fmt.Println(err)
		return nil, err
	}

	fs := osfs.New(j.GitPath)
	if _, err := fs.Stat(goGit.GitDirName); err == nil {
		fs, err = fs.Chroot(goGit.GitDirName)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
	}
	s := filesystem.NewStorageWithOptions(fs, gitCache.NewObjectLRUDefault(), filesystem.Options{KeepDescriptors: true})
	r, err := goGit.Open(s, fs)
	if err != nil {
		fmt.Println(err)
		return r, err
	}

	fmt.Println("repository cloned successfully")
	return r, nil
}

func getRepoCommits(r *goGit.Repository, since time.Time, until time.Time) ([]object.Commit, error) {
	ref, err := r.Head()
	if err != nil {
		return []object.Commit{}, err
	}
	cIter, err := r.Log(&goGit.LogOptions{
		From: ref.Hash(),
		// logOrder 4 is to sort commits desc order by commiter date
		Order: goGit.LogOrder(4),
		Since: &since,
		Until: &until,
	})

	commits := make([]object.Commit, 0)
	err = cIter.ForEach(func(c *object.Commit) error {
		commits = append(commits, *c)
		return nil
	})
	return commits, nil
}

func (j *DSGit) getFirstCommit(ctx *shared.Ctx, repo *goGit.Repository) (*object.Commit, error) {
	//git log --pretty=oneline --reverse | head -1
	cmd := fmt.Sprintf("cd %s; git log --pretty=oneline --reverse | head -1", j.GitPath)
	output, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return nil, err
	}

	fCommit := strings.Split(strings.TrimSpace(string(output)), " ")
	firstCommit, err := repo.CommitObject(plumbing.NewHash(fCommit[0]))
	if err != nil {
		return firstCommit, err
	}
	return firstCommit, nil
}

func (j *DSGit) getCloc(ctx *shared.Ctx, headSha string) error {
	cmdLine := []string{"cloc", "commit", headSha, "--json"}
	sout, serr, err := shared.ExecCommand(ctx, cmdLine, j.GitPath, GitDefaultEnv)
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "Sync"}).Errorf("error executing command: %v, error: %v, output: %s, output error: %s", cmdLine, err, sout, serr)
		return err
	}

	if sout == "" {
		return nil
	}

	r := make(map[string]clocResult)
	err = jsoniter.Unmarshal([]byte(sout), &r)
	if err != nil {
		j.log.WithFields(logrus.Fields{"operation": "Sync"}).Errorf("error unmarshall: %v, error: %v", sout, err)
		return err
	}
	j.headLinesOfCode = r["SUM"].Code
	return nil
}

// CommitCache single commit cache schema
type CommitCache struct {
	Timestamp      string `json:"timestamp"`
	EntityID       string `json:"entity_id"`
	SourceEntityID string `json:"source_entity_id"`
	FileLocation   string `json:"file_location"`
	Hash           string `json:"hash"`
	Orphaned       bool   `json:"orphaned"`
	FromDL         bool   `json:"from_dl"`
	Content        string `json:"content"`
	CommitDate     time.Time
}

// ReportData schema
type ReportData struct {
	ID              string `json:"id"`
	SfdcID          string `json:"sfdc_id"`
	ProjectName     string `json:"project_name"`
	URL             string `json:"url"`
	NewCommits      int64  `json:"new_commits"`
	Date            int64  `json:"date"`
	SyncStatus      string `json:"sync_status"`
	OrphanedCommits int64  `json:"orphaned_commits"`
}

type clocResult struct {
	Code          int `json:"code"`
	Blank         int `json:"blank"`
	Comment       int `json:"comment"`
	NumberOfFiles int `json:"nFiles"`
}

type lastSyncFile struct {
	LastSync      time.Time `json:"last_sync"`
	Target        int       `json:"target,omitempty"`
	Total         int       `json:"total,omitempty"`
	Head          string    `json:"head,omitempty"`
	FirstCommitAt time.Time `json:"first_commit_At"`
}

// CommitHashFields elected fields from commit schema to hash
type CommitHashFields struct {
	ID            string
	SHA           string
	RepositoryURL string
	RepositoryID  string
	Message       string
}
