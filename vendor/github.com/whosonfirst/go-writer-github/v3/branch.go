package writer

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const BRANCH_UUID_PREFIX string = "{prefix}-"

// AssignBranchPrefix tests 'branch' to see if it starts with the value of the `BRANCH_UUID_PREFIX` constant.
// If true it will replace that string with the value of the current Unix timestamp followed by "-" followed
// by a new UUID (v4) string. For example if 'branch' is "{prefix}-test" it would become "{TIMESTAMP}-{UUID}-test".
func AssignBranchPrefix(branch string) string {

	if !strings.HasPrefix(branch, BRANCH_UUID_PREFIX) {
		return branch
	}

	id := uuid.New()
	now := time.Now()

	prefix := fmt.Sprintf("%d-%s-", now.Unix(), id.String())
	return strings.Replace(branch, BRANCH_UUID_PREFIX, prefix, 1)
}
