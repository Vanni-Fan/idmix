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

static int values_equal(const idmix_value_t* a, const idmix_value_t* b, size_t n) {
    for (size_t i = 0; i < n; i++) {
        if (a[i].otype != b[i].otype || a[i].val != b[i].val) return 0;
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
    idmix_free_values(out);
    return ok;
}

static int decode_vector(idmix_ctx_t* ctx, const char* encoded, const idmix_value_t* want, size_t n,
                         const char* name) {
    idmix_value_t* out = NULL;
    size_t out_n = 0;
    int ok = idmix_decode(ctx, encoded, &out, &out_n) == 0 && out_n == n && values_equal(want, out, n);
    if (!ok) fprintf(stderr, "FAIL cross-language: %s\n", name);
    else printf("OK cross-language: %s\n", name);
    idmix_free_values(out);
    return ok;
}

int main(void) {
    srand((unsigned)time(NULL));
    idmix_ctx_t* ctx = idmix_create(NULL);
    if (!ctx) return 1;

    int ok = 1;

    idmix_value_t basic[] = {
        {IDMIX_OTYPE_UINT16, 5},
        {IDMIX_OTYPE_INT64, -1},
        {IDMIX_OTYPE_UINT32, 40},
    };
    ok &= round_trip(ctx, basic, 3, "basic");

    idmix_value_t u32max[] = {{IDMIX_OTYPE_UINT32, EXTREME_UINT32_MAX}};
    ok &= round_trip(ctx, u32max, 1, "uint32_max");

    idmix_value_t i32min[] = {{IDMIX_OTYPE_INT32, EXTREME_INT32_MIN}};
    ok &= round_trip(ctx, i32min, 1, "int32_min");

    idmix_value_t i64min[] = {{IDMIX_OTYPE_INT64, EXTREME_INT64_MIN}};
    ok &= round_trip(ctx, i64min, 1, "int64_min");

    idmix_value_t i64max[] = {{IDMIX_OTYPE_INT64, EXTREME_INT64_MAX}};
    ok &= round_trip(ctx, i64max, 1, "int64_max");

    idmix_value_t u64max[] = {{IDMIX_OTYPE_UINT64, (int64_t)-1}};
    ok &= round_trip(ctx, u64max, 1, "uint64_max");

    idmix_value_t mixed[] = {
        {IDMIX_OTYPE_UINT32, EXTREME_UINT32_MAX},
        {IDMIX_OTYPE_INT32, EXTREME_INT32_MIN},
        {IDMIX_OTYPE_INT64, EXTREME_INT64_MIN},
        {IDMIX_OTYPE_INT64, EXTREME_INT64_MAX},
    };
    ok &= round_trip(ctx, mixed, 4, "mixed_extremes");

    ok &= decode_vector(ctx, "hYpGvRq6B", basic, 3, "spec_example");
    ok &= decode_vector(ctx, "LwMDzFPIwK", u32max, 1, "uint32_max_vector");
    ok &= decode_vector(ctx, "LwMH4is20x", i32min, 1, "int32_min_vector");
    ok &= decode_vector(ctx, "eA3BqyCfeJ73bad1", i64min, 1, "int64_min_vector");
    ok &= decode_vector(ctx, "bTcNSaewCwrxPlc5fGCbq11xnBz120cpBTJ1A6ztNY", mixed, 4, "mixed_extremes_vector");

    idmix_destroy(ctx);
    if (!ok) return 1;
    printf("All tests passed.\n");
    return 0;
}
