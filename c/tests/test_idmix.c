#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "idmix.h"

#define EXTREME_UINT32_MAX 4294967295LL
#define EXTREME_INT32_MIN (-2147483648LL)
#define EXTREME_INT64_MIN INT64_MIN
#define EXTREME_INT64_MAX INT64_MAX

static idmix_value_t str_val(const char* s) {
    idmix_value_t v = {IDMIX_KIND_STRING, 0, 0, (char*)s, strlen(s)};
    return v;
}

static int values_equal(const idmix_value_t* a, const idmix_value_t* b, size_t n) {
    for (size_t i = 0; i < n; i++) {
        if (a[i].kind != b[i].kind) return 0;
        if (a[i].kind == IDMIX_KIND_STRING) {
            if (a[i].str_len != b[i].str_len) return 0;
            if (memcmp(a[i].str, b[i].str, a[i].str_len) != 0) return 0;
        } else {
            if (a[i].otype != b[i].otype || a[i].val != b[i].val) return 0;
        }
    }
    return 1;
}

static int round_trip(idmix_ctx_t* ctx, const idmix_value_t* in, size_t n, const char* name) {
    char* s = NULL;
    if (idmix_encode(ctx, in, n, &s) != 0) return 0;
    idmix_value_t* out = NULL;
    size_t out_n = 0;
    int ok = idmix_decode(ctx, s, &out, &out_n) == 0 && out_n == n && values_equal(in, out, n);
    if (!ok) fprintf(stderr, "FAIL round trip: %s\n", name);
    else printf("OK round trip: %s\n", name);
    idmix_free_string(s);
    idmix_free_values(out, out_n);
    return ok;
}

static int decode_vector(idmix_ctx_t* ctx, const char* encoded, const idmix_value_t* want, size_t n,
                         const char* name) {
    idmix_value_t* out = NULL;
    size_t out_n = 0;
    int ok = idmix_decode(ctx, encoded, &out, &out_n) == 0 && out_n == n && values_equal(want, out, n);
    if (!ok) fprintf(stderr, "FAIL cross-language: %s\n", name);
    else printf("OK cross-language: %s\n", name);
    idmix_free_values(out, out_n);
    return ok;
}

static int encode_deterministic(idmix_ctx_t* ctx, int variant, const idmix_value_t* in, size_t n,
                                const char* want, const char* name) {
    char* s = NULL;
    int ok = idmix_encode_with_variant(ctx, variant, in, n, &s) == 0 && s && strcmp(s, want) == 0;
    if (!ok) fprintf(stderr, "FAIL re-encode: %s (got %s, want %s)\n", name, s ? s : "(null)", want);
    else printf("OK re-encode: %s\n", name);
    idmix_free_string(s);
    return ok;
}

int main(void) {
    srand((unsigned)time(NULL));
    idmix_ctx_t* ctx = idmix_create(NULL);
    if (!ctx) return 1;

    int ok = 1;

    idmix_value_t basic[] = {
        IDMIX_INT(IDMIX_OTYPE_UINT16, 5),
        IDMIX_INT(IDMIX_OTYPE_INT64, -1),
        IDMIX_INT(IDMIX_OTYPE_UINT32, 40),
    };
    ok &= round_trip(ctx, basic, 3, "basic");

    idmix_value_t u32max[] = {IDMIX_INT(IDMIX_OTYPE_UINT32, EXTREME_UINT32_MAX)};
    ok &= round_trip(ctx, u32max, 1, "uint32_max");

    idmix_value_t i32min[] = {IDMIX_INT(IDMIX_OTYPE_INT32, EXTREME_INT32_MIN)};
    ok &= round_trip(ctx, i32min, 1, "int32_min");

    idmix_value_t i64min[] = {IDMIX_INT(IDMIX_OTYPE_INT64, EXTREME_INT64_MIN)};
    ok &= round_trip(ctx, i64min, 1, "int64_min");

    idmix_value_t i64max[] = {IDMIX_INT(IDMIX_OTYPE_INT64, EXTREME_INT64_MAX)};
    ok &= round_trip(ctx, i64max, 1, "int64_max");

    idmix_value_t u64max[] = {IDMIX_INT(IDMIX_OTYPE_UINT64, (int64_t)-1)};
    ok &= round_trip(ctx, u64max, 1, "uint64_max");

    idmix_value_t mixed[] = {
        IDMIX_INT(IDMIX_OTYPE_UINT32, EXTREME_UINT32_MAX),
        IDMIX_INT(IDMIX_OTYPE_INT32, EXTREME_INT32_MIN),
        IDMIX_INT(IDMIX_OTYPE_INT64, EXTREME_INT64_MIN),
        IDMIX_INT(IDMIX_OTYPE_INT64, EXTREME_INT64_MAX),
    };
    ok &= round_trip(ctx, mixed, 4, "mixed_extremes");

    idmix_value_t string_ex[] = {
        str_val("hello"),
        IDMIX_INT(IDMIX_OTYPE_UINT16, 5),
        str_val("\xe4\xb8\x96\xe7\x95\x8c"),
    };
    ok &= round_trip(ctx, string_ex, 3, "string_example");

    ok &= decode_vector(ctx, "ixHjl0FK7", basic, 3, "spec_example");
    ok &= decode_vector(ctx, "hUdZLNKGa", u32max, 1, "uint32_max_vector");
    ok &= decode_vector(ctx, "hUdElRoHP", i32min, 1, "int32_min_vector");
    ok &= decode_vector(ctx, "8B10qg6x0EAf3b", i64min, 1, "int64_min_vector");
    ok &= decode_vector(ctx, "8B2cU8kbWpQ2RM", i64max, 1, "int64_max_vector");
    ok &= decode_vector(ctx, "8B3CPRsv0Owa6S", u64max, 1, "uint64_max_vector");
    ok &= decode_vector(ctx, "bULoRnNZJinEZGKD78wIigIaw6QplS8B0HGNCKO2L6", mixed, 4, "mixed_extremes_vector");
    ok &= decode_vector(ctx, "ceOqw5RPaTfgnfXyp7Sdepb", string_ex, 3, "string_example_vector");

    ok &= encode_deterministic(ctx, 0, basic, 3, "ixHjl0FK7", "spec_example_reencode");
    ok &= encode_deterministic(ctx, 0, string_ex, 3, "ceOqw5RPaTfgnfXyp7Sdepb", "string_example_reencode");

    uint8_t raw[] = {0x08, 0x96, 0x01};
    char* text = NULL;
    uint8_t* back = NULL;
    size_t back_len = 0;
    ok &= idmix_encode_bytes(raw, sizeof(raw), &text) == 0;
    ok &= idmix_decode_string(text, &back, &back_len) == 0 && back_len == sizeof(raw) &&
          memcmp(back, raw, sizeof(raw)) == 0;
    if (ok) printf("OK encode_bytes/decode_string\n");
    else fprintf(stderr, "FAIL encode_bytes/decode_string\n");
    idmix_free_string(text);
    idmix_codec_free_bytes(back);

    idmix_destroy(ctx);
    if (!ok) return 1;
    printf("All tests passed.\n");
    return 0;
}
