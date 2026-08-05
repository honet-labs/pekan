-- Migration: 0039_whatsapp_bot_queue.sql
-- Description: Create table for WhatsApp chatbot asynchronous message queue and statistics

CREATE TABLE IF NOT EXISTS public.whatsapp_bot_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    reply_message TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'success', 'failed'
    error_message TEXT,
    processing_time_ms INT,
    tenant_id UUID,
    user_id UUID,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    
    FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE SET NULL,
    FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_bot_queue_status ON public.whatsapp_bot_queue(status);
CREATE INDEX IF NOT EXISTS idx_whatsapp_bot_queue_received_at ON public.whatsapp_bot_queue(received_at);
CREATE INDEX IF NOT EXISTS idx_whatsapp_bot_queue_phone ON public.whatsapp_bot_queue(phone_number);
