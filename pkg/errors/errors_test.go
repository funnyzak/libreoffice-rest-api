package errors

import "testing"

func TestDomainErrorType(t *testing.T) {
	t.Parallel()

	err := NewValidation("参数错误", "缺少字段", nil)
	if !IsType(err, ErrorValidation) {
		t.Fatalf("期望错误类型为 validation")
	}

	var de *DomainError
	if !AsDomainError(err, &de) {
		t.Fatalf("期望识别为领域错误")
	}
}
