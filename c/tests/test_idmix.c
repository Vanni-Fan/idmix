#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "idmix.h"

static int check_bin(void) {
    idmix_ctx_t* ctx = idmix_create(NULL);
    if (!ctx) return 1;
    idmix_value_t vals[] = {
        {IDMIX_OTYPE_UINT16, 5},
        {IDMIX_OTYPE_INT64, -1},
        {IDMIX_OTYPE_UINT32, 40},
    };
    char* s = NULL;
    if (idmix_encode(ctx, vals, 3, &s) != 0) {
        idmix_destroy(ctx);
        return 1;
    }
    idmix_value_t* out = NULL;
    size_t n = 0;
    if (idmix_decode(ctx, s, &out, &n) != 0 || n != 3) {
        idmix_free_string(s);
        idmix_destroy(ctx);
        return 1;
    }
    int ok = out[0].val == 5 && out[1].val == -1 && out[2].val == 40;
    idmix_free_string(s);
    idmix_free_values(out);
    idmix_destroy(ctx);
    printf("%s round trip basic\n", ok ? "OK" : "FAIL");
    return ok ? 0 : 1;
}

int main(void) {
    srand((unsigned)time(NULL));
    return check_bin();
}
