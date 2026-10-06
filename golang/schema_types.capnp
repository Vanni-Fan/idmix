# 用于 IDX vs Cap'n Proto 基准对比的 Cap'n Proto schema。
# 定义 TypedPair 结构及 TypedPairList（含 pairs 列表）。
#
# 生成 Go 代码（需要 capnp 工具）：
#   capnp compile -ogo schema_types.capnp

@0xd4a8e7f9c1b3e6a0;

struct TypedPair {
    otype @0 :Int32;
    val   @1 :Int64;
}

struct TypedPairList {
    pairs @0 :List(TypedPair);
}
