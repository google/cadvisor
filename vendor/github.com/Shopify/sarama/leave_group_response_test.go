package sarama

import "testing"

var (
	leaveGroupResponseNoError   = []byte{0x00, 0x00}
	leaveGroupResponseWithError = []byte{0, 25}
)

func TestLeaveGroupResponse(t *testing.T) {
	var response *LeaveGroupResponse

	response = new(LeaveGroupResponse)
	testDecodable(t, "no error", response, leaveGroupResponseNoError)
	if response.Err != ErrNoError {
		t.Error("Decoding error failed: no error expected but found", response.Err)
	}

	response = new(LeaveGroupResponse)
	testDecodable(t, "with error", response, leaveGroupResponseWithError)
	if response.Err != ErrUnknownMemberId {
		t.Error("Decoding error failed: ErrUnknownMemberId expected but found", response.Err)
	}
}
