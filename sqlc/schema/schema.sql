DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_type t
        JOIN pg_namespace n ON n.oid = t.typnamespace
    WHERE t.typname = 'status_enum'
        AND n.nspname = 'public'
) THEN CREATE TYPE public.status_enum AS ENUM (
    'ACTIVE',
    'INACTIVE',
    'SUSPENDED',
    'DELETED'
);
END IF;
END $$;
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    email VARCHAR(150) NOT NULL UNIQUE,
    username VARCHAR(50) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'USER',
    status status_enum NOT NULL DEFAULT 'ACTIVE',
    company_id UUID NULL,
    department_id UUID NULL,
    last_login_at TIMESTAMPTZ NULL,
    created_by UUID NULL,
    updated_by UUID NULL,
    deleted_by UUID NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL
);
CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    trade_name VARCHAR(150),
    document VARCHAR(20) UNIQUE,
    document_type VARCHAR(10),
    email VARCHAR(150),
    phone VARCHAR(30),
    website VARCHAR(150),
    address_street VARCHAR(150),
    address_number VARCHAR(20),
    address_complement VARCHAR(100),
    address_neighborhood VARCHAR(100),
    address_city VARCHAR(100),
    address_state VARCHAR(2),
    address_zipcode VARCHAR(20),
    address_country VARCHAR(50) DEFAULT 'BR',
    status status_enum NOT NULL DEFAULT 'ACTIVE',
    timezone VARCHAR(50) DEFAULT 'America/Sao_Paulo',
    created_by UUID NULL,
    updated_by UUID NULL,
    deleted_by UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL
);
CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(255),
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
    created_by UUID NULL,
    updated_by UUID NULL,
    deleted_by UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT fk_departments_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT uq_department_name_per_company UNIQUE (company_id, name)
);
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    category_id UUID NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    barcode VARCHAR(50),
    quantity INT NOT NULL DEFAULT 0,
    size VARCHAR(50),
    cost_price NUMERIC(10, 2),
    sale_price NUMERIC(10, 2),
    created_by UUID NULL,
    updated_by UUID NULL,
    deleted_by UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT fk_products_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES product_categories(id) ON DELETE
    SET NULL,
        CONSTRAINT uq_product_name_per_company UNIQUE (company_id, name)
);
CREATE TABLE IF NOT EXISTS product_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7) DEFAULT '#FFFFFF',
    status status_enum NOT NULL DEFAULT 'ACTIVE',
    created_by UUID NULL,
    updated_by UUID NULL,
    deleted_by UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT fk_product_categories_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT uq_product_category_name_per_company UNIQUE (company_id, name)
);
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_type
    WHERE typname = 'gender_enum'
) THEN CREATE TYPE gender_enum AS ENUM ('MALE', 'FEMALE', 'OTHER', 'NOT_SAY');
END IF;
END $$;
CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    birth_date DATE NOT NULL,
    cpf VARCHAR(14) NOT NULL,
    rg VARCHAR(20),
    marital_status VARCHAR(20),
    gender gender_enum DEFAULT 'NOT_SAY',
    whatsapp VARCHAR(20),
    mobile_phone VARCHAR(20),
    home_phone VARCHAR(20),
    email VARCHAR(100) NOT NULL,
    address_street VARCHAR(150),
    address_number VARCHAR(20),
    address_complement VARCHAR(100),
    address_neighborhood VARCHAR(100),
    address_city VARCHAR(100),
    address_state VARCHAR(2),
    address_zipcode VARCHAR(20),
    address_country VARCHAR(50) DEFAULT 'BR',
    balance_due DECIMAL(10, 2) DEFAULT 0,
    created_by UUID NULL,
    updated_by UUID NULL,
    deleted_by UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT fk_customers_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT uq_customer_cpf_company UNIQUE (company_id, cpf),
    CONSTRAINT uq_customer_email_company UNIQUE (company_id, email)
);
-- Status da Conta/Venda (sales)
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_type
    WHERE typname = 'sale_status_enum'
) THEN CREATE TYPE sale_status_enum AS ENUM (
    'pending',
    'paid',
    'overdue',
    'scheduled',
    'canceled'
);
END IF;
END $$;
-- Métodos de Pagamento
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_type
    WHERE typname = 'payment_method_enum'
) THEN CREATE TYPE payment_method_enum AS ENUM (
    'cash',
    'credit_card',
    'debit_card',
    'pix',
    'bank_transfer',
    'installments',
    'other'
);
END IF;
END $$;
CREATE TABLE IF NOT EXISTS sales (
    -- Identificadores (UUID)
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    company_id UUID NOT NULL,
    -- Informações da Venda
    -- Mudei para TIMESTAMPTZ para registrar hora exata da transação
    sale_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    discount_amount NUMERIC(10, 2) DEFAULT 0.00,
    subtotal NUMERIC(10, 2) NOT NULL,
    total_amount NUMERIC(10, 2) NOT NULL,
    down_payment NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    installments_count INTEGER NOT NULL DEFAULT 1,
    due_days INT DEFAULT NULL,
    payment_method payment_method_enum DEFAULT 'cash',
    status account_status_enum DEFAULT 'pending',
    -- Auditoria (Traceability) usando TIMESTAMPTZ
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID DEFAULT NULL,
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    deleted_by UUID DEFAULT NULL,
    -- Constraints
    CONSTRAINT fk_sale_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE RESTRICT,
    CONSTRAINT fk_sale_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS sale_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sale_id UUID NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(10, 2) NOT NULL,
    discount DECIMAL(10, 2) DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    type payment_method_enum NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_payment_methods_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS vendors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    name VARCHAR(200) NOT NULL,
    tax_id VARCHAR(18),
    email VARCHAR(100),
    phone VARCHAR(20),
    postal_code VARCHAR(20),
    address_line_1 VARCHAR(255),
    address_line_2 VARCHAR(255),
    number VARCHAR(20),
    neighborhood VARCHAR(100),
    city VARCHAR(100),
    state VARCHAR(50),
    country VARCHAR(100) DEFAULT 'Brazil',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_vendors_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS bill_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_bill_categories_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT uq_bill_categories_name_company UNIQUE (company_id, name)
);
CREATE TABLE IF NOT EXISTS bills_payable (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    vendor_id UUID,
    category_id UUID,
    payment_method_id UUID,
    amount DECIMAL(12, 2) NOT NULL,
    due_date DATE NOT NULL,
    status bill_status_enum DEFAULT 'pending',
    description TEXT,
    scheduled_date DATE,
    payment_date DATE,
    amount_paid DECIMAL(12, 2),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_bills_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT fk_bills_vendor FOREIGN KEY (vendor_id) REFERENCES vendors(id) ON DELETE
    SET NULL,
        CONSTRAINT fk_bills_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE
    SET NULL,
        CONSTRAINT fk_bills_payment_method FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id) ON DELETE
    SET NULL
);
CREATE TABLE IF NOT EXISTS payment_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    sale_id UUID,
    payment_method_id UUID,
    user_id UUID NOT NULL,
    amount_paid DECIMAL(12, 2) NOT NULL,
    payment_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    notes TEXT,
    CONSTRAINT fk_payment_history_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT fk_payment_history_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
    CONSTRAINT fk_payment_history_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_payment_history_sale FOREIGN KEY (sale_id) REFERENCES sales(id) ON DELETE
    SET NULL,
        CONSTRAINT fk_payment_history_method FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id) ON DELETE
    SET NULL
);
CREATE TABLE IF NOT EXISTS accounts_receivable (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    sale_id UUID NOT NULL,
    total_amount NUMERIC(10, 2) NOT NULL,
    balance NUMERIC(10, 2) NOT NULL,
    due_date DATE NOT NULL,
    installment_number INT DEFAULT 1,
    total_installments INT DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_receivable_company FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT fk_receivable_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE RESTRICT,
    CONSTRAINT fk_receivable_sale FOREIGN KEY (sale_id) REFERENCES sales(id) ON DELETE CASCADE
);
CREATE INDEX idx_receivable_customer ON accounts_receivable(customer_id);
CREATE INDEX idx_receivable_status ON accounts_receivable(status);
CREATE INDEX idx_receivable_due_date ON accounts_receivable(due_date);