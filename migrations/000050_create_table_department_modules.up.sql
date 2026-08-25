CREATE TABLE department_modules (
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    module_code VARCHAR NOT NULL REFERENCES modules(code),
    PRIMARY KEY (department_id, module_code)
);