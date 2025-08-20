-- SQL для створення таблиці в Supabase (спрощена версія)
-- Запустіть цей код в SQL Editor вашого Supabase проекту

-- Спочатку видаліть таблицю, якщо вона існує
DROP TABLE IF EXISTS zvit_table;

-- Створення таблиці без CHECK constraints
CREATE TABLE zvit_table (
    id BIGSERIAL PRIMARY KEY,
    fio TEXT NOT NULL,
    ipn TEXT NOT NULL,
    organization TEXT NOT NULL,
    date_opened DATE NOT NULL,
    date_first_deposit DATE,
    account_status TEXT NOT NULL DEFAULT 'Очікує активацію',
    card_status TEXT NOT NULL DEFAULT 'На випуску',
    documents JSONB DEFAULT '{"contract": false, "passport": false, "questionnaire": false}',
    comment TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Вимкнення RLS для спрощення
ALTER TABLE zvit_table DISABLE ROW LEVEL SECURITY;

-- Надання повних прав для anon користувача
GRANT ALL ON TABLE zvit_table TO anon;
GRANT ALL ON TABLE zvit_table TO authenticated;

-- Надання прав на sequence
GRANT USAGE, SELECT ON SEQUENCE zvit_table_id_seq TO anon;
GRANT USAGE, SELECT ON SEQUENCE zvit_table_id_seq TO authenticated;

-- Функція для автоматичного визначення статусу рахунку
CREATE OR REPLACE FUNCTION update_account_status()
RETURNS TRIGGER AS $$
BEGIN
  -- Автоматично встановлюємо статус на основі дати першого зарахування
  IF NEW.date_first_deposit IS NOT NULL THEN
    NEW.account_status = 'Активний';
  ELSE
    NEW.account_status = 'Очікує активацію';
  END IF;
  
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Тригер для автоматичного оновлення статусу при вставці та оновленні
CREATE TRIGGER trigger_update_account_status
  BEFORE INSERT OR UPDATE ON zvit_table
  FOR EACH ROW
  EXECUTE FUNCTION update_account_status();
