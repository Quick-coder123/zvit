-- SQL для створення таблиці в Supabase
-- Запустіть цей код в SQL Editor вашого Supabase проекту

CREATE TABLE zvit_table (
    id SERIAL PRIMARY KEY,
    fio TEXT NOT NULL,
    ipn TEXT NOT NULL,
    organization TEXT NOT NULL,
    date_opened DATE NOT NULL,
    date_first_deposit DATE,
    account_status TEXT NOT NULL CHECK (account_status IN ('активний', 'заблокований', 'закритий')),
    card_status TEXT NOT NULL CHECK (card_status IN ('активна', 'заблокована', 'не випущена')),
    documents TEXT,
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Створення RLS (Row Level Security) політик
ALTER TABLE zvit_table ENABLE ROW LEVEL SECURITY;

-- Політика для читання (дозволити всім читати)
CREATE POLICY "Enable read access for all users" ON zvit_table
    FOR SELECT USING (true);

-- Політика для вставки (дозволити всім додавати)
CREATE POLICY "Enable insert for all users" ON zvit_table
    FOR INSERT WITH CHECK (true);

-- Політика для оновлення (дозволити всім оновлювати)
CREATE POLICY "Enable update for all users" ON zvit_table
    FOR UPDATE USING (true);

-- Політика для видалення (дозволити всім видаляти)
CREATE POLICY "Enable delete for all users" ON zvit_table
    FOR DELETE USING (true);
