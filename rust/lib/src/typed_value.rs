use crate::error::IdMixError;

pub const OTYPE_UINT8: u8 = 0;
pub const OTYPE_UINT16: u8 = 1;
pub const OTYPE_UINT32: u8 = 2;
pub const OTYPE_UINT64: u8 = 3;
pub const OTYPE_INT8: u8 = 4;
pub const OTYPE_INT16: u8 = 5;
pub const OTYPE_INT32: u8 = 6;
pub const OTYPE_INT64: u8 = 7;

/// 编解码时保留原始类型的整数值。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct TypedValue {
    pub otype: u8,
    pub val: i64,
}

/// 对外暴露的带类型整数值。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Value {
    U8(u8),
    U16(u16),
    U32(u32),
    U64(u64),
    I8(i8),
    I16(i16),
    I32(i32),
    I64(i64),
}

impl Value {
    pub fn to_typed(self) -> Result<TypedValue, IdMixError> {
        match self {
            Value::U8(v) => Ok(TypedValue {
                otype: OTYPE_UINT8,
                val: v as i64,
            }),
            Value::U16(v) => Ok(TypedValue {
                otype: OTYPE_UINT16,
                val: v as i64,
            }),
            Value::U32(v) => Ok(TypedValue {
                otype: OTYPE_UINT32,
                val: v as i64,
            }),
            Value::U64(v) => {
                if v > i64::MAX as u64 {
                    return Err(IdMixError::msg(format!(
                        "uint64 value {v} overflows i64"
                    )));
                }
                Ok(TypedValue {
                    otype: OTYPE_UINT64,
                    val: v as i64,
                })
            }
            Value::I8(v) => Ok(TypedValue {
                otype: OTYPE_INT8,
                val: v as i64,
            }),
            Value::I16(v) => Ok(TypedValue {
                otype: OTYPE_INT16,
                val: v as i64,
            }),
            Value::I32(v) => Ok(TypedValue {
                otype: OTYPE_INT32,
                val: v as i64,
            }),
            Value::I64(v) => Ok(TypedValue {
                otype: OTYPE_INT64,
                val: v,
            }),
        }
    }

    pub fn from_typed(tv: TypedValue) -> Result<Self, IdMixError> {
        match tv.otype {
            OTYPE_UINT8 => Ok(Value::U8(tv.val as u8)),
            OTYPE_UINT16 => Ok(Value::U16(tv.val as u16)),
            OTYPE_UINT32 => Ok(Value::U32(tv.val as u32)),
            OTYPE_UINT64 => Ok(Value::U64(tv.val as u64)),
            OTYPE_INT8 => Ok(Value::I8(tv.val as i8)),
            OTYPE_INT16 => Ok(Value::I16(tv.val as i16)),
            OTYPE_INT32 => Ok(Value::I32(tv.val as i32)),
            OTYPE_INT64 => Ok(Value::I64(tv.val)),
            _ => Err(IdMixError::msg(format!("invalid otype {}", tv.otype))),
        }
    }
}

pub fn normalize_values(values: &[Value]) -> Result<Vec<TypedValue>, IdMixError> {
    values
        .iter()
        .copied()
        .map(Value::to_typed)
        .collect()
}

pub fn materialize_values(typed: &[TypedValue]) -> Result<Vec<Value>, IdMixError> {
    typed.iter().copied().map(Value::from_typed).collect()
}
