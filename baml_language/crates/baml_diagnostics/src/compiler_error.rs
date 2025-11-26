pub mod error_format;
pub mod parse_error;
pub mod type_error;

pub use parse_error::ParseError;
pub use type_error::TypeError;

use ariadne::{Report, ReportKind};
use baml_base::Span;

/// Every compiler error that can occur in the compiler.
/// It is parameterized by several types that are owned by the different compiler phases,
/// which we don't want to collect in this module. We only care that those values can
/// be displayed (for rendering in the message).
pub enum CompilerError<Ty> {
    ParseError(ParseError),
    TypeError(TypeError<Ty>),
}

pub struct ErrorCode(u32);

/// Error codes are formated like E0001, (E followed by a number padded to 4 digits).
impl std::fmt::Display for ErrorCode {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> Result<(), std::fmt::Error> {
        write!(f, "E{:04}", self.0)
    }
}

#[derive(Clone, Debug, PartialEq)]
pub enum ColorMode {
    Color,
    NoColor,
}

pub fn render_error<'a, Ty: std::fmt::Display>(
    color_mode: ColorMode,
    err: CompilerError<Ty>,
) -> Report<'a, Span> {
    let (report_builder, code) = error_format::error_report_and_code(err);
    report_builder
        .with_config(ariadne::Config::default().with_color(color_mode == ColorMode::Color))
        .with_note(format!("Error code: {code}"))
        .finish()
}

const TYPE_MISMATCH: ErrorCode = ErrorCode(1);
const UNKNOWN_TYPE: ErrorCode = ErrorCode(2);
const UNKNOWN_VARIABLE: ErrorCode = ErrorCode(3);
const INVALID_OPERATOR: ErrorCode = ErrorCode(4);
const ARGUMENT_COUNT_MISMATCH: ErrorCode = ErrorCode(5);
const NOT_CALLABLE: ErrorCode = ErrorCode(6);
const NO_SUCH_FIELD: ErrorCode = ErrorCode(7);
const NOT_INDEXABLE: ErrorCode = ErrorCode(8);

const UNEXPECTED_EOF: ErrorCode = ErrorCode(9);
const UNEXPECTED_TOKEN: ErrorCode = ErrorCode(10);
