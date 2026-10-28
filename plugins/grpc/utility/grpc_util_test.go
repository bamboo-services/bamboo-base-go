package xGrpcUtil

import (
	"context"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	xGrpcConst "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2/constant"
	"google.golang.org/grpc/metadata"
)

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		key         xGrpcConst.Metadata
		wantVal     string
		wantErrCode apperr.Code
		wantErrMsg  string
		secretVal   string
	}{
		{
			name:        "nil context",
			ctx:         nil,
			key:         xGrpcConst.MetadataRequestUUID,
			wantVal:     "",
			wantErrCode: 40000,
			wantErrMsg:  "缺少 gRPC 传入元数据",
		},
		{
			name:        "context.Background without metadata",
			ctx:         context.Background(),
			key:         xGrpcConst.MetadataRequestUUID,
			wantVal:     "",
			wantErrCode: 40000,
			wantErrMsg:  "缺少 gRPC 传入元数据",
		},
		{
			name: "missing key in metadata",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				xGrpcConst.MetadataAppAccessID.String(), "app-123",
			)),
			key:         xGrpcConst.MetadataRequestUUID,
			wantVal:     "",
			wantErrCode: 40000,
			wantErrMsg:  "元数据中不存在有效值: x-request-uuid",
		},
		{
			name: "only whitespace value",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				xGrpcConst.MetadataAppSecretKey.String(), "   \t \n ",
			)),
			key:         xGrpcConst.MetadataAppSecretKey,
			wantVal:     "",
			wantErrCode: 40000,
			wantErrMsg:  "元数据中不存在有效值: app-secret-key",
			secretVal:   "super-secret-must-not-leak",
		},
		{
			name: "multivalue returns first valid trimmed",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				xGrpcConst.MetadataRequestUUID.String(), "   ",
				xGrpcConst.MetadataRequestUUID.String(), "  trace-id-abc  ",
				xGrpcConst.MetadataRequestUUID.String(), "second-id",
			)),
			key:         xGrpcConst.MetadataRequestUUID,
			wantVal:     "trace-id-abc",
			wantErrCode: 0,
		},
		{
			name: "single value trimmed successfully",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				xGrpcConst.MetadataRequestUUID.String(), "\t req-999 \n",
			)),
			key:         xGrpcConst.MetadataRequestUUID,
			wantVal:     "req-999",
			wantErrCode: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotVal, gotErr := ExtractMetadata(tc.ctx, tc.key)
			if tc.wantErrCode == 0 {
				if gotErr != nil {
					t.Fatalf("expected nil error, got: %v", gotErr)
				}
				if gotVal != tc.wantVal {
					t.Fatalf("expected value %q, got %q", tc.wantVal, gotVal)
				}
			} else {
				if gotErr == nil {
					t.Fatalf("expected error code %d, got nil error", tc.wantErrCode)
				}
				if gotErr.Code.GetCode() != tc.wantErrCode {
					t.Fatalf("expected error code %d, got %d", tc.wantErrCode, gotErr.Code.GetCode())
				}
				if tc.wantErrMsg != "" && gotErr.ErrorMessage.String() != tc.wantErrMsg {
					t.Fatalf("expected error msg %q, got %q", tc.wantErrMsg, gotErr.ErrorMessage.String())
				}
				if tc.secretVal != "" {
					if strings.Contains(gotErr.Error(), tc.secretVal) || strings.Contains(gotErr.ErrorMessage.String(), tc.secretVal) {
						t.Fatalf("error leaked secret value")
					}
				}
			}
		})
	}
}
