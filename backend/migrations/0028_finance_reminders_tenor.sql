-- Add tenor fields to reminders table
ALTER TABLE finance_reminders 
ADD COLUMN IF NOT EXISTS total_tenor INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS current_tenor INTEGER DEFAULT 0;
