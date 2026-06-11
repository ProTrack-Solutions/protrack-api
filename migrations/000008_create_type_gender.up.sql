DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_type
    WHERE typname = 'gender_enum'
) THEN CREATE TYPE gender_enum AS ENUM ('MALE', 'FEMALE', 'OTHER', 'NOT_SAY');
END IF;
END $$;