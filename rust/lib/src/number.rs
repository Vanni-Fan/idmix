use crate::error::IdMixError;
use crate::idx_codec::{DataObject, MAX_STRING_LEN};
use crate::typed_value::{
    OTYPE_INT16, OTYPE_INT32, OTYPE_INT64, OTYPE_INT8, OTYPE_UINT16, OTYPE_UINT32, OTYPE_UINT64,
    OTYPE_UINT8, Value,
};

pub fn normalize_objects(values: &[Value]) -> Result<Vec<DataObject>, IdMixError> {
    values.iter().map(object_from_value).collect()
}

pub fn object_from_value(v: &Value) -> Result<DataObject, IdMixError> {
    match v {
        Value::String(s) => {
            if s.is_empty() {
                return Err(IdMixError::msg(format!(
                    "empty string is not allowed (max {MAX_STRING_LEN} bytes)"
                )));
            }
            if s.len() > MAX_STRING_LEN {
                return Err(IdMixError::msg(format!(
                    "string length {} exceeds max {MAX_STRING_LEN}",
                    s.len()
                )));
            }
            Ok(DataObject::string(s.as_bytes()))
        }
        Value::Bytes(b) => {
            if b.is_empty() {
                return Err(IdMixError::msg(format!(
                    "empty byte slice is not allowed (max {MAX_STRING_LEN} bytes)"
                )));
            }
            if b.len() > MAX_STRING_LEN {
                return Err(IdMixError::msg(format!(
                    "byte slice length {} exceeds max {MAX_STRING_LEN}",
                    b.len()
                )));
            }
            Ok(DataObject::string(b))
        }
        Value::U8(v) => Ok(DataObject::integer(OTYPE_UINT8, *v as i64)),
        Value::U16(v) => Ok(DataObject::integer(OTYPE_UINT16, *v as i64)),
        Value::U32(v) => Ok(DataObject::integer(OTYPE_UINT32, *v as i64)),
        Value::U64(v) => Ok(DataObject::integer(OTYPE_UINT64, *v as i64)),
        Value::I8(v) => Ok(DataObject::integer(OTYPE_INT8, *v as i64)),
        Value::I16(v) => Ok(DataObject::integer(OTYPE_INT16, *v as i64)),
        Value::I32(v) => Ok(DataObject::integer(OTYPE_INT32, *v as i64)),
        Value::I64(v) => Ok(DataObject::integer(OTYPE_INT64, *v)),
    }
}

pub fn materialize_objects(objects: &[DataObject]) -> Result<Vec<Value>, IdMixError> {
    objects.iter().map(materialize_object).collect()
}

fn materialize_object(obj: &DataObject) -> Result<Value, IdMixError> {
    if obj.is_string {
        return Ok(Value::String(
            String::from_utf8(obj.str.clone()).map_err(|e| IdMixError::msg(e.to_string()))?,
        ));
    }
    materialize_value(obj.otype, obj.val)
}

fn materialize_value(otype: u8, val: i64) -> Result<Value, IdMixError> {
    match otype {
        OTYPE_UINT8 => Ok(Value::U8(val as u8)),
        OTYPE_UINT16 => Ok(Value::U16(val as u16)),
        OTYPE_UINT32 => Ok(Value::U32(val as u32)),
        OTYPE_UINT64 => Ok(Value::U64(val as u64)),
        OTYPE_INT8 => Ok(Value::I8(val as i8)),
        OTYPE_INT16 => Ok(Value::I16(val as i16)),
        OTYPE_INT32 => Ok(Value::I32(val as i32)),
        OTYPE_INT64 => Ok(Value::I64(val)),
        _ => Err(IdMixError::msg(format!("invalid otype {otype}"))),
    }
}
