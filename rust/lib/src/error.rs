use thiserror::Error;

#[derive(Debug, Error)]
pub enum IdMixError {
    #[error("{0}")]
    Message(String),
}

impl IdMixError {
    pub fn msg(s: impl Into<String>) -> Self {
        Self::Message(s.into())
    }
}
