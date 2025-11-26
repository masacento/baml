// ============================================================================
// Type Errors
// ============================================================================
//
use baml_base::{Name, Span};

/// Type errors that can occur during type checking.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum TypeError<T> {
    /// Type mismatch between expected and found types.
    TypeMismatch { expected: T, found: T, span: Span },
    /// Reference to an unknown type name.
    UnknownType { name: String, span: Span },
    /// Reference to an unknown variable.
    UnknownVariable { name: String, span: Span },
    /// Invalid binary operation.
    InvalidBinaryOp {
        op: String,
        lhs: T,
        rhs: T,
        span: Span,
    },
    /// Invalid unary operation.
    InvalidUnaryOp { op: String, operand: T, span: Span },
    /// Wrong number of arguments in function call.
    ArgumentCountMismatch {
        expected: usize,
        found: usize,
        span: Span,
    },
    /// Calling a non-callable type.
    NotCallable { ty: T, span: Span },
    /// Field access on non-class type.
    NoSuchField { ty: T, field: String, span: Span },
    /// Index access on non-indexable type.
    NotIndexable { ty: T, span: Span },
}
