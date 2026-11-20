package valueerr_test

import (
	"fmt"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/valueerr"
)

func Example_noCause() {
	validateReceiver := func(receiver *string) *apperr.Error {
		if receiver == nil {
			return apperr.New(valueerr.InvalidReceiver, "转换接收器不能为空")
		}
		return nil
	}

	appErr := validateReceiver(nil)
	fmt.Printf("%d %s %t\n", appErr.Code.Code, appErr.Code.Output, appErr.Err == nil)
	// Output: 50019 VALUE_INVALID_RECEIVER true
}
