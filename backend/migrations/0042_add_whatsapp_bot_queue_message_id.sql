-- Migration: 0042_add_whatsapp_bot_queue_message_id.sql
-- Description: Add message_id to whatsapp_bot_queue with unique constraint to prevent duplicate processing

ALTER TABLE public.whatsapp_bot_queue ADD COLUMN IF NOT EXISTS message_id VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_bot_queue_message_id 
ON public.whatsapp_bot_queue(message_id) 
WHERE message_id IS NOT NULL;
