# Padeir0's Intermediate Representation

Intermediate representation meant to be used in my compilers.

This repository is not tested directly, tests are made through
Millipascal's frontend, using it's test suite.


Goals:

 - [x] Have unsigned integers (`u8`, `u16`, `u32`, `u64`)
 - [x] Have bitwise operations (left-shift, right-shift, bitwise-or, bitwise-and, bitwise-not, bitwise-xor)
 - [ ] Have special read-only pointers to the stack top and frame init
 - [ ] Allow metadata to be stored in stack frames
 - [ ] Have malloc/free builtins (not dependent on C)
 - [ ] Have floats (`f32`, `f64`)
 - [ ] Have FFI to C procedures
 - [ ] Compile to JS or WASM
