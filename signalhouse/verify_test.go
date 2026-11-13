package signalhouse

import (
	"context"
	"strings"
	"testing"
)

const verifyPhone = "+12025550123"

func TestVerifyTransport(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	verify := client.Verify

	if _, err := verify.CheckSimSwap(ctx, CheckSimSwapParams{PhoneNumber: verifyPhone, MaxAge: 240, IncludeSwapDate: true}, &RequestOptions{IdempotencyKey: "k1"}); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/verify/sim-swap" || last.header.Get("Idempotency-Key") != "k1" {
		t.Fatalf("sim swap contract: %s %s", last.method, last.uri)
	}
	if string(mustJSON(t, last.body)) != `{"includeSwapDate":true,"maxAge":240,"phoneNumber":"+12025550123"}` {
		t.Fatalf("sim swap body: %s", mustJSON(t, last.body))
	}
	if _, err := verify.CheckSimSwap(ctx, CheckSimSwapParams{PhoneNumber: verifyPhone}, nil); err != nil {
		t.Fatal(err)
	}
	if string(mustJSON(t, last.body)) != `{"phoneNumber":"+12025550123"}` {
		t.Fatalf("sim swap body omits unset fields: %s", mustJSON(t, last.body))
	}
	if _, err := verify.CheckSimSwap(ctx, CheckSimSwapParams{}, nil); err == nil || !strings.Contains(err.Error(), "phoneNumber") {
		t.Fatalf("missing phoneNumber must fail before transport: %v", err)
	}

	if _, err := verify.StartNumberVerification(ctx, StartNumberVerificationParams{
		PhoneNumber: verifyPhone, DevicePhoneNumber: verifyPhone, RedirectURL: "https://example.com/done",
	}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/verify/number" {
		t.Fatalf("number verify contract: %s %s", last.method, last.uri)
	}
	if string(mustJSON(t, last.body)) != `{"devicePhoneNumber":"+12025550123","phoneNumber":"+12025550123","redirectUrl":"https://example.com/done"}` {
		t.Fatalf("number verify body: %s", mustJSON(t, last.body))
	}
	if _, err := verify.StartNumberVerification(ctx, StartNumberVerificationParams{PhoneNumber: verifyPhone}, nil); err != nil {
		t.Fatal(err)
	}
	if string(mustJSON(t, last.body)) != `{"phoneNumber":"+12025550123"}` {
		t.Fatalf("number verify body omits unset fields: %s", mustJSON(t, last.body))
	}
	if _, err := verify.StartNumberVerification(ctx, StartNumberVerificationParams{}, nil); err == nil || !strings.Contains(err.Error(), "phoneNumber") {
		t.Fatalf("missing phoneNumber must fail before transport: %v", err)
	}

	if _, err := verify.GetNumberVerification(ctx, GetNumberVerificationParams{CheckID: "NVC1/2"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/verify/number/NVC1%2F2" || last.body != nil {
		t.Fatalf("read contract: %s %s %v", last.method, last.uri, last.body)
	}
	if _, err := verify.GetNumberVerification(ctx, GetNumberVerificationParams{}, nil); err == nil || !strings.Contains(err.Error(), "checkId") {
		t.Fatalf("missing checkId must fail before transport: %v", err)
	}
}
