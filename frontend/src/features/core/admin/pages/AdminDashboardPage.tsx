import { useEffect, useState, useRef } from "react";
import * as echarts from "echarts";
import { 
  adminLogin, adminLogout, bootstrapTenant, listLogs, listTenants, 
  getGrowthStats, updateTenantQuotas, listTenantModules, updateTenantModule,
  updateTenant, deleteTenant,
  getGlobalSetting, saveGlobalSetting as saveAdminGlobalSetting,
  impersonateUser, getServerStatus, testNotification, testAI, testDatabase,
  listBackups, createBackup, restoreBackup, downloadBackupBlob,
  executeQuery, getDatabaseStats, getDatabaseGrowth, listTenantUsers,
  adminResetUserPassword, adminUpdateUserEmail, adminUpdateUserPhone,
  listTenantBackups, createTenantBackup, restoreTenantBackup, downloadTenantBackupBlob,
  getWhatsAppQueueStats, getWhatsAppQueueHistory, retryWhatsAppQueueMessage,
  checkUpdate, applyUpdate, getUpdateStatus,
  TenantListItem, AuditLog, GrowthStats, TenantModule, ServerStatus, BackupFile, DatabaseTable, DatabaseGrowthPoint, QueryResult, TenantUser,
  WhatsAppQueueStats, WhatsAppQueueItem, UpdateStatusInfo, UpdateProgress
} from "../api/admin.api";

import { useToast } from "../../../../core/hooks/useToast";
import { ToastContainer } from "../../../../core/components/Toast";
import { useI18n } from "../../../../core/i18n/i18n";
import { useDebounce } from "../../../../core/hooks/useDebounce";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";
import { PasswordStrength } from "../../../../core/components/PasswordStrength";
import { BackToTop } from "../../../../core/components/BackToTop";
import { PasswordInput } from "../../../../core/components/PasswordInput";
import { PageHeader } from "../../../../core/components/PageHeader";

type Tab = "dashboard" | "tenants" | "add_tenant" | "stats" | "server" | "logs" | "notifications" | "ai" | "database" | "backups" | "storage" | "optimization" | "dbtool" | "whatsapp" | "updates";

const DEFAULT_WA_BOT_SYSTEM_PROMPT = `Anda adalah Asisten AI PEKAN, perencana keuangan pribadi yang profesional, ringkas, dan sangat membantu.
Tugas Anda adalah membalas pesan pengguna WhatsApp secara interaktif. Pengguna sudah login/terverifikasi.

--- ATURAN BERKOMUNIKASI ---
1. Jawablah menggunakan bahasa Indonesia yang natural, profesional, sopan, dan langsung pada intinya (to the point).
2. HINDARI mengulang-ulang sapaan formal pembuka yang sama (seperti "Halo! Selamat siang/sore/malam. Senang sekali bisa membantu..." atau "Sebagai Asisten AI PEKAN...") di setiap pesan. Langsung jawab pertanyaan pengguna secara spesifik.
3. Jika pengguna menyapa singkat (seperti 'halo' atau 'hai'), sapa balik secara singkat, bersahabat, dan ingatkan secara ringkas bahwa Anda dapat membantu mencatat transaksi (misal: 'catat pengeluaran bensin 20rb') atau membacakan laporan keuangan.
4. Jika pengguna menanyakan sisa anggaran, pengeluaran, pemasukan, atau laporan transaksi, bacakan data rill di bawah ini secara akurat. Tampilkan data dengan rapi menggunakan poin-poin terstruktur agar mudah dibaca.
5. Berikan saran atau rekomendasi finansial secara cerdas, realistis, dan memotivasi tanpa menggurui.
6. Gunakan format tebal (bold) WhatsApp dengan tanda bintang (*) untuk hal-hal penting seperti kategori, nominal rupiah, atau sisa anggaran agar nyaman dibaca di layar HP.
7. Jangan menyebutkan bahwa Anda adalah model bahasa besar. Berperanlah 100% sebagai Asisten AI PEKAN.`;

export function AdminDashboardPage(): JSX.Element {
  const { t, locale, setLocale } = useI18n();
  const { toasts, success, error, remove } = useToast();
  
  const [isLoggedIn, setIsLoggedIn] = useState(!!localStorage.getItem("pekan_admin_token"));
  const [secret, setSecret] = useState("");
  const [activeTab, setActiveTab] = useState<Tab>((localStorage.getItem("admin_active_tab") as Tab) || "dashboard");
  
  const [loading, setLoading] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [profileMenuOpen, setProfileMenuOpen] = useState(false);
  const [tenantsExpanded, setTenantsExpanded] = useState(true);
  const [systemExpanded, setSystemExpanded] = useState(true);
  
  const [tenants, setTenants] = useState<TenantListItem[]>([]);
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [stats, setStats] = useState<GrowthStats | null>(null);
  const [server, setServer] = useState<ServerStatus | null>(null);
  const [backups, setBackups] = useState<BackupFile[]>([]);
  const [backupType, setBackupType] = useState("full");
  const [isBackingUp, setIsBackingUp] = useState(false);
  const [fileToRestore, setFileToRestore] = useState<string | null>(null);
  const [isRestoring, setIsRestoring] = useState(false);

  const [searchTerm, setSearchTerm] = useState("");
  const debouncedSearch = useDebounce(searchTerm, 500);

  const [dateFrom, setDateFrom] = useState(() => { const d = new Date(); d.setDate(d.getDate() - 30); return d.toISOString().slice(0, 10); });
  const [dateTo, setDateTo] = useState(() => new Date().toISOString().slice(0, 10));
  const debouncedDateFrom = useDebounce(dateFrom, 500);
  const debouncedDateTo = useDebounce(dateTo, 500);
  const [selectedTenantID, setSelectedTenantID] = useState<string | null>(null);
  const [selectedModules, setSelectedModules] = useState<TenantModule[]>([]);
  const [editingTenant, setEditingTenant] = useState<TenantListItem | null>(null);
  const [tenantUsers, setTenantUsers] = useState<TenantUser[]>([]);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [tenantBackups, setTenantBackups] = useState<BackupFile[]>([]);
  const [loadingBackups, setLoadingBackups] = useState(false);
  const [tenantModalTab, setTenantModalTab] = useState<'info' | 'backups' | 'modules'>('info');
  const [userToReset, setUserToReset] = useState<{ user: TenantUser, action: 'password' | 'email' | 'phone' } | null>(null);
  const [resetValue, setResetValue] = useState("");
  const [isResetting, setIsResetting] = useState(false);
  const [tenantToDelete, setTenantToDelete] = useState<TenantListItem | null>(null);

  // WhatsApp Bot Queue States
  const [waQueueStats, setWaQueueStats] = useState<WhatsAppQueueStats | null>(null);
  const [waQueueHistory, setWaQueueHistory] = useState<WhatsAppQueueItem[]>([]);
  const [waChartData, setWaChartData] = useState<WhatsAppQueueItem[]>([]);
  const [waQueueTotal, setWaQueueTotal] = useState(0);
  const [waQueueLimit, setWaQueueLimit] = useState(10);
  const [waQueueOffset, setWaQueueOffset] = useState(0);
  const [waQueueSearch, setWaQueueSearch] = useState("");
  const debouncedWaSearch = useDebounce(waQueueSearch, 500);
  const [loadingWaQueue, setLoadingWaQueue] = useState(false);
  const [retryingMsgId, setRetryingMsgId] = useState<string | null>(null);

  // New Filters & Auto-Refresh States
  const [waDateRange, setWaDateRange] = useState("all"); // "all" | "today" | "7days" | "30days" | "custom"
  const [waStartDate, setWaStartDate] = useState("");
  const [waEndDate, setWaEndDate] = useState("");
  const [waAutoRefresh, setWaAutoRefresh] = useState("off"); // "off" | "1m" | "5m"
  const waRefreshTimer = useRef<any>(null);

  // System Update States
  const [updateInfo, setUpdateInfo] = useState<UpdateStatusInfo | null>(null);
  const [updateProgress, setUpdateProgress] = useState<UpdateProgress | null>(null);
  const [checkingUpdate, setCheckingUpdate] = useState(false);
  const [applyingUpdate, setApplyingUpdate] = useState(false);
  const updatePollTimer = useRef<any>(null);
  const terminalLogEndRef = useRef<HTMLDivElement>(null);

  const [form, setForm] = useState({
    tenant_code: "",
    tenant_name: "",
    admin_email: "owner@pekan.local",
    admin_name: "Workspace Owner",
    password: "password"
  });

  const chartRef = useRef<HTMLDivElement>(null);
  const statsTenantsRef = useRef<HTMLDivElement>(null);
  const statsUsersRef = useRef<HTMLDivElement>(null);
  const dbGrowthRef = useRef<HTMLDivElement>(null);
  const queueChartRef = useRef<HTMLDivElement>(null);
  const statusChartRef = useRef<HTMLDivElement>(null);
  const latencyChartRef = useRef<HTMLDivElement>(null);
  const refreshInterval = useRef<any>(null);


  // Notification Config States
  const [smtpConfig, setSmtpConfig] = useState({ host: "", port: "", username: "", password: "", security: "none" });
  const [telegramConfig, setTelegramConfig] = useState({ botToken: "", botName: "" });
  const [waConfig, setWaConfig] = useState({ apiToken: "", phoneId: "" });
  const [waFonnteConfig, setWaFonnteConfig] = useState({ apiKey: "" });
  const [waWahaConfig, setWaWahaConfig] = useState({ apiUrl: "", apiKey: "", session: "default" });
  const [waGowaConfig, setWaGowaConfig] = useState({ apiKey: "" });
  const [geminiConfig, setGeminiConfig] = useState({ apiKey: "", model: "gemini-2.0-flash" });
  const [openaiConfig, setOpenaiConfig] = useState({ apiKey: "", model: "gpt-4o-mini" });
  const [claudeConfig, setClaudeConfig] = useState({ apiKey: "", model: "claude-3-5-sonnet-20240620" });
  const [sumopodConfig, setSumopodConfig] = useState({ apiKey: "", model: "sumopod-v1" });
  const [activeAI, setActiveAI] = useState("gemini");
  const [activeWaBotAI, setActiveWaBotAI] = useState("gemini");
  const [waActiveProvider, setWaActiveProvider] = useState("wa_waha");
  const [waBotSystemPrompt, setWaBotSystemPrompt] = useState("");
  const [waBotPhoneNumber, setWaBotPhoneNumber] = useState("");

  // Track whether each AI key is already saved in DB (masked on read)
  const [geminiKeySaved, setGeminiKeySaved] = useState(false);
  const [openaiKeySaved, setOpenaiKeySaved] = useState(false);
  const [claudeKeySaved, setClaudeKeySaved] = useState(false);
  const [sumopodKeySaved, setSumopodKeySaved] = useState(false);
  
  const [dbConfig, setDbConfig] = useState({ host: "127.0.0.1", port: "5432", user: "postgres", password: "postgres", dbname: "pekan", sslmode: "disable" });
  
  // Storage Config States
  const [storageActiveProvider, setStorageActiveProvider] = useState("local");
  const [s3Config, setS3Config] = useState({ region: "us-east-1", bucket: "", accessKey: "", secretKey: "", endpoint: "" });
  const [gdriveConfig, setGdriveConfig] = useState({ folderId: "", credentialsJson: "" });
  const [localConfig, setLocalConfig] = useState({ path: "./data/storage" });
  const [s3KeySaved, setS3KeySaved] = useState(false);
  const [gdriveKeySaved, setGdriveKeySaved] = useState(false);

  const [dbStats, setDbStats] = useState<DatabaseTable[]>([]);
  const [dbGrowth, setDbGrowth] = useState<DatabaseGrowthPoint[]>([]);
  const [sqlQuery, setSqlQuery] = useState("SELECT relname, n_live_tup FROM pg_stat_user_tables ORDER BY n_live_tup DESC LIMIT 10;");

  const [queryResult, setQueryResult] = useState<QueryResult | null>(null);
  const [isExecutingQuery, setIsExecutingQuery] = useState(false);
  const [isMasked, setIsMasked] = useState(true);
  const [optConfig, setOptConfig] = useState({ api_rate_limit: "2000", api_timeout: "30s", max_upload_size: "10mb" });

  const [registrationEnabled, setRegistrationEnabled] = useState(true);
  const [registrationOTPMethod, setRegistrationOTPMethod] = useState("email");
  
  const [aiModels, setAiModels] = useState<{id: string, label: string}[]>(() => {
    const saved = localStorage.getItem("admin_ai_models");
    try {
      return saved ? (JSON.parse(saved) || []) : [
        { id: "gemini-2.0-flash", label: "Gemini 2.0 Flash" },
        { id: "gemini-2.0-pro", label: "Gemini 2.0 Pro" },
        { id: "gemini-2.5-flash", label: "Gemini 2.5 Flash" },
        { id: "gemini-3.0-flash", label: "Gemini 3.0 Flash" },
        { id: "gemini-3.0-pro", label: "Gemini 3.0 Pro" }
      ];
    } catch (e) { return []; }
  });
  const [openaiModels, setOpenaiModels] = useState<{id: string, label: string}[]>(() => {
    const saved = localStorage.getItem("admin_openai_models");
    try {
      return saved ? (JSON.parse(saved) || []) : [
        { id: "gpt-4o-mini", label: "GPT-4o Mini" },
        { id: "gpt-4o", label: "GPT-4o" },
        { id: "gpt-4-turbo", label: "GPT-4 Turbo" },
        { id: "gpt-3.5-turbo", label: "GPT-3.5 Turbo" }
      ];
    } catch (e) { return []; }
  });
  const [claudeModels, setClaudeModels] = useState<{id: string, label: string}[]>(() => {
    const saved = localStorage.getItem("admin_claude_models");
    try {
      return saved ? (JSON.parse(saved) || []) : [
        {id: "claude-3-5-sonnet-20240620", label: "Claude 3.5 Sonnet"},
        {id: "claude-3-5-sonnet-latest", label: "Claude 3.5 Sonnet (Latest)"},
        {id: "claude-3-5-haiku-20241022", label: "Claude 3.5 Haiku"},
        {id: "claude-3-opus-20240229", label: "Claude 3 Opus"},
        {id: "claude-3-sonnet-20240229", label: "Claude 3 Sonnet"},
        {id: "claude-3-haiku-20240307", label: "Claude 3 Haiku"},
      ];
    } catch (e) { return []; }
  });
  const [sumopodModels, setSumopodModels] = useState<{id: string, label: string}[]>(() => {
    const saved = localStorage.getItem("admin_sumopod_models");
    try {
      return saved ? (JSON.parse(saved) || []) : [];
    } catch (e) { return []; }
  });
  
  const [savingConfig, setSavingConfig] = useState<"smtp" | "telegram" | "wa" | "wa_fonnte" | "wa_waha" | "wa_gowa" | "gemini" | "openai" | "claude" | "sumopod" | "active_ai" | "active_wa_bot_ai" | "wa_active_provider" | "wa_bot_system_prompt" | "wa_bot_phone_number" | "database" | null>(null);
  const [testingConfig, setTestingConfig] = useState<string | null>(null);

  // Test Connection Modal States
  const [showTestModal, setShowTestModal] = useState(false);
  const [testProvider, setTestProvider] = useState<string>("");
  const [testDestination, setTestDestination] = useState("");

  const loadGlobalSettings = async () => {
    const fetchSetting = async (key: string) => {
      try {
        const data = await getGlobalSetting(key);
        return data;
      } catch (e) { /* ignore */ }
      return null;
    };

    // AI settings — for API keys: only update state if we have a real value (not masked).
    // If masked (is_masked=true), the key is already saved; track this and show a placeholder.
    const gKey = await fetchSetting("receipt_api_key_gemini");
    const gModel = await fetchSetting("receipt_model_gemini");
    setGeminiKeySaved(!!gKey?.is_masked);
    setGeminiConfig(prev => ({
      apiKey: "", // always clear field; user must re-enter to change
      model: (gModel && gModel.value) ? gModel.value : prev.model,
    }));

    const oKey = await fetchSetting("receipt_api_key_openai");
    const oModel = await fetchSetting("receipt_model_openai");
    setOpenaiKeySaved(!!oKey?.is_masked);
    setOpenaiConfig(prev => ({
      apiKey: "",
      model: (oModel && oModel.value) ? oModel.value : prev.model,
    }));

    const cKey = await fetchSetting("receipt_api_key_claude");
    const cModel = await fetchSetting("receipt_model_claude");
    setClaudeKeySaved(!!cKey?.is_masked);
    setClaudeConfig(prev => ({
      apiKey: "",
      model: (cModel && cModel.value) ? cModel.value : prev.model,
    }));

    const sKey = await fetchSetting("receipt_api_key_sumopod");
    const sModel = await fetchSetting("receipt_model_sumopod");
    setSumopodKeySaved(!!sKey?.is_masked);
    setSumopodConfig(prev => ({
      apiKey: "",
      model: (sModel && sModel.value) ? sModel.value : prev.model,
    }));

    const activeAIVal = await fetchSetting("receipt_active_ai_provider");
    if (activeAIVal && activeAIVal.value) setActiveAI(activeAIVal.value);

    const activeWaBotAIVal = await fetchSetting("wa_bot_active_ai_provider");
    if (activeWaBotAIVal && activeWaBotAIVal.value) setActiveWaBotAI(activeWaBotAIVal.value);

    const waPromptVal = await fetchSetting("wa_bot_system_instructions");
    if (waPromptVal && waPromptVal.value) {
      setWaBotSystemPrompt(waPromptVal.value);
    } else {
      setWaBotSystemPrompt(DEFAULT_WA_BOT_SYSTEM_PROMPT);
    }

    const waBotPhoneVal = await fetchSetting("wa_bot_phone_number");
    if (waBotPhoneVal && waBotPhoneVal.value) {
      setWaBotPhoneNumber(waBotPhoneVal.value);
    }

    const smtpVal = await fetchSetting("notification_smtp");
    if (smtpVal && smtpVal.value) { 
      try { 
        const parsed = JSON.parse(smtpVal.value);
        setSmtpConfig({
          ...parsed,
          security: parsed.security || "none"
        }); 
      } catch {/* */} 
    }

    const tgVal = await fetchSetting("notification_telegram");
    if (tgVal && tgVal.value) { try { setTelegramConfig(JSON.parse(tgVal.value)); } catch {/* */} }

    const waVal = await fetchSetting("notification_wa");
    if (waVal && waVal.value) { try { setWaConfig(JSON.parse(waVal.value)); } catch {/* */} }

    const waFonnteVal = await fetchSetting("notification_wa_fonnte");
    if (waFonnteVal && waFonnteVal.value) { try { setWaFonnteConfig(JSON.parse(waFonnteVal.value)); } catch {/* */} }

    const waWahaVal = await fetchSetting("notification_wa_waha");
    if (waWahaVal && waWahaVal.value) { try { setWaWahaConfig(JSON.parse(waWahaVal.value)); } catch {/* */} }

    const waGowaVal = await fetchSetting("notification_wa_gowa");
    if (waGowaVal && waGowaVal.value) { try { setWaGowaConfig(JSON.parse(waGowaVal.value)); } catch {/* */} }

    const waActiveProvVal = await fetchSetting("notification_wa_active_provider");
    if (waActiveProvVal && waActiveProvVal.value) setWaActiveProvider(waActiveProvVal.value);

    const dbVal = await fetchSetting("database_config");
    if (dbVal && dbVal.value) { try { setDbConfig(JSON.parse(dbVal.value)); } catch {/* */} }

    const storageProvVal = await fetchSetting("storage_active_provider");
    if (storageProvVal && storageProvVal.value) setStorageActiveProvider(storageProvVal.value);

    const s3Val = await fetchSetting("storage_s3_config");
    if (s3Val && s3Val.value) { 
      try { 
        const parsed = JSON.parse(s3Val.value);
        setS3Config({ ...parsed, secretKey: "" }); 
        setS3KeySaved(true);
      } catch {/* */} 
    }

    const gdriveVal = await fetchSetting("storage_gdrive_config");
    if (gdriveVal && gdriveVal.value) { 
      try { 
        const parsed = JSON.parse(gdriveVal.value);
        setGdriveConfig({ ...parsed, credentialsJson: "" }); 
        setGdriveKeySaved(true);
      } catch {/* */} 
    }

    const optVal = await fetchSetting("optimization_config");
    if (optVal && optVal.value) { try { setOptConfig(JSON.parse(optVal.value)); } catch {/* */} }
  };

  useEffect(() => {
    if (isLoggedIn) {
      // Always load stats for summary cards if on dashboard
      if (activeTab === "dashboard") {
        loadStats();
        loadServer();
      }
      
      if (activeTab === "tenants") loadTenants();
      if (activeTab === "logs") loadLogs();
      if (activeTab === "stats") loadStats();
      if (activeTab === "backups") loadBackups();
      if (activeTab === "server") {
        loadServer();
        refreshInterval.current = setInterval(loadServer, 5000);
      } else {
        if (refreshInterval.current) clearInterval(refreshInterval.current);
      }
      if (activeTab === "notifications" || activeTab === "database" || activeTab === "ai" || activeTab === "storage" || activeTab === "optimization") {
        loadGlobalSettings();
      }
      if (activeTab === "dbtool") {
        loadDbStats();
        loadDbGrowth();
      }
      if (activeTab === "whatsapp") {
        loadWhatsAppQueueStats();
      }
      if (activeTab === "updates") {
        handleCheckUpdate();
        pollUpdateStatus();
        updatePollTimer.current = setInterval(pollUpdateStatus, 2500);
      } else {
        if (updatePollTimer.current) clearInterval(updatePollTimer.current);
      }

    }
    return () => {
      if (refreshInterval.current) clearInterval(refreshInterval.current);
      if (updatePollTimer.current) clearInterval(updatePollTimer.current);
    };
  }, [isLoggedIn, activeTab]);

  useEffect(() => {
    if (activeTab === "whatsapp" && waAutoRefresh !== "off") {
      const intervalMs = waAutoRefresh === "1m" ? 60000 : 300000;
      waRefreshTimer.current = setInterval(() => {
        loadWhatsAppQueueStats();
        loadWhatsAppQueueHistory(waQueueLimit, waQueueOffset, waQueueSearch);
      }, intervalMs);
      return () => {
        if (waRefreshTimer.current) {
          clearInterval(waRefreshTimer.current);
        }
      };
    }
  }, [activeTab, waAutoRefresh, waQueueLimit, waQueueOffset, waQueueSearch]);

  useEffect(() => {
    if (editingTenant?.id) {
      const loadData = async () => {
        setLoadingUsers(true);
        setLoadingBackups(true);
        try {
          const [users, backups] = await Promise.all([
            listTenantUsers(editingTenant.id),
            listTenantBackups(editingTenant.id)
          ]);
          setTenantUsers(users);
          setTenantBackups(backups);
        } catch (err) {
          error("Gagal memuat data tenant.");
        } finally {
          setLoadingUsers(false);
          setLoadingBackups(false);
        }
      };
      loadData();
    } else {
      setTenantUsers([]);
      setTenantBackups([]);
    }
  }, [editingTenant?.id]);

  useEffect(() => {
    if (activeTab === "dbtool" && dbGrowthRef.current && dbGrowth.length > 0) {
      const chart = echarts.init(dbGrowthRef.current);
      
      // Process data: group by date
      const dates = Array.from(new Set(dbGrowth.map(p => p.date.split('T')[0]))).sort();
      const schemas = Array.from(new Set(dbGrowth.map(p => p.schema_name)));
      
      const series = schemas.map(schema => {
        return {
          name: schema,
          type: 'line',
          smooth: true,
          data: dates.map(date => {
            const point = dbGrowth.find(p => p.date.startsWith(date) && p.schema_name === schema);
            return point ? (point.total_size_bytes / 1024 / 1024).toFixed(2) : null;
          })
        };
      });

      const option = {
        title: { text: 'Database Growth (MB)', left: 'center', textStyle: { color: '#64748b', fontSize: 14 } },
        tooltip: { 
          trigger: 'axis',
          formatter: (params: any[]) => {
            let res = `<div style="font-weight: 600; margin-bottom: 4px;">${params[0].axisValue}</div>`;
            params.forEach(item => {
              res += `<div style="display: flex; justify-content: space-between; gap: 20px;">
                <span>${item.marker} ${item.seriesName}</span>
                <span style="font-weight: 600;">${item.value} MB</span>
              </div>`;
            });
            return res;
          }
        },
        legend: { 
          bottom: 0, 
          type: 'scroll',
          textStyle: { color: '#64748b', fontSize: 10 } 
        },
        grid: { top: 60, left: 60, right: 30, bottom: 80 },
        xAxis: { 
          type: 'category', 
          data: dates,
          axisLabel: { color: '#64748b', fontSize: 10 }
        },
        yAxis: { 
          type: 'value',
          axisLabel: { 
            color: '#64748b', 
            fontSize: 10,
            formatter: '{value} MB'
          },
          splitLine: { lineStyle: { color: '#e2e8f0', opacity: 0.5 } }
        },
        series: series
      };

      chart.setOption(option);
      const handleResize = () => chart.resize();
      window.addEventListener("resize", handleResize);
      return () => {
        window.removeEventListener("resize", handleResize);
        chart.dispose();
      };
    }
  }, [activeTab, dbGrowth]);

  useEffect(() => {
    localStorage.setItem("admin_active_tab", activeTab);
  }, [activeTab]);

  useEffect(() => {
    if (activeTab === "dashboard" && stats && chartRef.current) {
      const chart = echarts.init(chartRef.current);
      const option = {
        tooltip: { 
          trigger: 'axis',
          formatter: (params: any[]) => {
            let res = `<div style="font-weight: 600; margin-bottom: 4px;">Date: ${params[0].axisValue}</div>`;
            params.forEach(item => {
              res += `<div style="display: flex; justify-content: space-between; gap: 20px;">
                <span>${item.marker} ${item.seriesName}</span>
                <span style="font-weight: 600;">${item.value}</span>
              </div>`;
            });
            return res;
          }
        },
        legend: { data: [t("admin.nav_tenants"), t("common.user")], bottom: 0, textStyle: { color: '#64748b' } },
        grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true },
        xAxis: {
          type: 'category',
          boundaryGap: false,
          data: (stats?.tenants || []).map(t => t.date.slice(5)),
          axisLabel: { color: '#64748b' }
        },
        yAxis: { 
          type: 'value',
          axisLabel: { color: '#64748b' }
        },
        series: [
          {
            name: t("admin.nav_tenants"),
            type: 'line',
            smooth: true,
            areaStyle: { opacity: 0.2 },
            itemStyle: { color: '#2dd4bf' },
            data: (stats?.tenants || []).map(t => t.count)
          },
          {
            name: t("common.user"),
            type: 'line',
            smooth: true,
            areaStyle: { opacity: 0.1 },
            itemStyle: { color: '#fbbf24' },
            data: (stats?.users || []).map(u => u.count)
          }
        ]
      };
      chart.setOption(option);
      const handleResize = () => chart.resize();
      window.addEventListener('resize', handleResize);
      return () => {
        window.removeEventListener('resize', handleResize);
        chart.dispose();
      };
    }
  }, [activeTab, stats]);

  useEffect(() => {
    const filterByDateRange = (items: WhatsAppQueueItem[]) => {
      if (waDateRange === "all") return items;
      const now = new Date();
      return items.filter(item => {
        const itemDate = new Date(item.received_at);
        if (waDateRange === "today") {
          return itemDate.toDateString() === now.toDateString();
        }
        if (waDateRange === "7days") {
          const diffTime = Math.abs(now.getTime() - itemDate.getTime());
          const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
          return diffDays <= 7;
        }
        if (waDateRange === "30days") {
          const diffTime = Math.abs(now.getTime() - itemDate.getTime());
          const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
          return diffDays <= 30;
        }
        if (waDateRange === "custom") {
          if (waStartDate) {
            const start = new Date(waStartDate);
            start.setHours(0,0,0,0);
            if (itemDate < start) return false;
          }
          if (waEndDate) {
            const end = new Date(waEndDate);
            end.setHours(23,59,59,999);
            if (itemDate > end) return false;
          }
          return true;
        }
        return true;
      });
    };

    const filteredChartData = filterByDateRange(waChartData);

    if (activeTab === "whatsapp" && filteredChartData.length > 0) {
      let qChart: any = null;
      let sChart: any = null;
      let lChart: any = null;

      // Group data points chronologically (oldest to newest)
      const sortedData = [...filteredChartData].reverse();
      
      const getChartGroups = (data: WhatsAppQueueItem[]) => {
        const groups: { [key: string]: { pending: number; processing: number; success: number; failed: number; total_latency: number; latency_count: number } } = {};
        
        let useMinutes = false;
        if (data.length > 0) {
          const oldest = new Date(data[0].received_at).getTime();
          const newest = new Date(data[data.length - 1].received_at).getTime();
          const diffHours = (newest - oldest) / (1000 * 60 * 60);
          if (diffHours < 2) {
            useMinutes = true;
          }
        }

        data.forEach(item => {
          const dateObj = new Date(item.received_at);
          let key = "";
          if (useMinutes) {
            const tens = Math.floor(dateObj.getMinutes() / 10) * 10;
            key = dateObj.toLocaleDateString("id-ID", { day: '2-digit', month: 'short' }) + " " + 
                  String(dateObj.getHours()).padStart(2, '0') + ":" + String(tens).padStart(2, '0');
          } else {
            key = dateObj.toLocaleDateString("id-ID", { day: '2-digit', month: 'short' }) + " " + 
                  String(dateObj.getHours()).padStart(2, '0') + ":00";
          }

          if (!groups[key]) {
            groups[key] = { pending: 0, processing: 0, success: 0, failed: 0, total_latency: 0, latency_count: 0 };
          }

          if (item.status === 'pending') groups[key].pending++;
          else if (item.status === 'processing') groups[key].processing++;
          else if (item.status === 'success') {
            groups[key].success++;
            if (item.processing_time_ms) {
              groups[key].total_latency += item.processing_time_ms;
              groups[key].latency_count++;
            }
          }
          else if (item.status === 'failed') {
            groups[key].failed++;
            if (item.processing_time_ms) {
              groups[key].total_latency += item.processing_time_ms;
              groups[key].latency_count++;
            }
          }
        });

        return groups;
      };

      const groups = getChartGroups(sortedData);
      const categories = Object.keys(groups);

      const pendingData = categories.map(c => groups[c].pending);
      const processingData = categories.map(c => groups[c].processing);
      const successData = categories.map(c => groups[c].success);
      const failedData = categories.map(c => groups[c].failed);
      const latencyData = categories.map(c => {
        const g = groups[c];
        return g.latency_count > 0 ? parseFloat((g.total_latency / g.latency_count / 1000).toFixed(2)) : 0;
      });

      // 1. Antrean & Proses AI Chart
      if (queueChartRef.current) {
        qChart = echarts.init(queueChartRef.current);
        const option = {
          tooltip: {
            trigger: 'axis',
            formatter: (params: any[]) => {
              let res = `<div style="font-weight: 600; margin-bottom: 4px;">Waktu: ${params[0].axisValue}</div>`;
              params.forEach(item => {
                res += `<div style="display: flex; justify-content: space-between; gap: 20px;">
                  <span>${item.marker} ${item.seriesName}</span>
                  <span style="font-weight: 600;">${item.value} Pesan</span>
                </div>`;
              });
              return res;
            }
          },
          legend: { 
            data: ["Sedang Diproses (Active AI)", "Dalam Antrean (Pending)"], 
            bottom: 0, 
            textStyle: { color: '#64748b' } 
          },
          grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true },
          xAxis: {
            type: 'category',
            boundaryGap: false,
            data: categories,
            axisLabel: { color: '#64748b' }
          },
          yAxis: {
            type: 'value',
            minInterval: 1,
            axisLabel: { color: '#64748b' }
          },
          series: [
            {
              name: "Sedang Diproses (Active AI)",
              type: 'line',
              smooth: true,
              areaStyle: { opacity: 0.15 },
              itemStyle: { color: '#4f46e5' },
              data: processingData
            },
            {
              name: "Dalam Antrean (Pending)",
              type: 'line',
              smooth: true,
              areaStyle: { opacity: 0.15 },
              itemStyle: { color: '#f59e0b' },
              data: pendingData
            }
          ]
        };
        qChart.setOption(option);
      }

      // 2. Status Respon Chart
      if (statusChartRef.current) {
        sChart = echarts.init(statusChartRef.current);
        const option = {
          tooltip: {
            trigger: 'axis',
            formatter: (params: any[]) => {
              let res = `<div style="font-weight: 600; margin-bottom: 4px;">Waktu: ${params[0].axisValue}</div>`;
              params.forEach(item => {
                res += `<div style="display: flex; justify-content: space-between; gap: 20px;">
                  <span>${item.marker} ${item.seriesName}</span>
                  <span style="font-weight: 600;">${item.value} Pesan</span>
                </div>`;
              });
              return res;
            }
          },
          legend: { 
            data: ["Sukses", "Gagal"], 
            bottom: 0, 
            textStyle: { color: '#64748b' } 
          },
          grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true },
          xAxis: {
            type: 'category',
            boundaryGap: false,
            data: categories,
            axisLabel: { color: '#64748b' }
          },
          yAxis: {
            type: 'value',
            minInterval: 1,
            axisLabel: { color: '#64748b' }
          },
          series: [
            {
              name: "Sukses",
              type: 'line',
              smooth: true,
              areaStyle: { opacity: 0.15 },
              itemStyle: { color: '#10b981' },
              data: successData
            },
            {
              name: "Gagal",
              type: 'line',
              smooth: true,
              areaStyle: { opacity: 0.15 },
              itemStyle: { color: '#ef4444' },
              data: failedData
            }
          ]
        };
        sChart.setOption(option);
      }

      // 3. Rata-rata Latensi AI Respon Chart
      if (latencyChartRef.current) {
        lChart = echarts.init(latencyChartRef.current);
        const option = {
          tooltip: {
            trigger: 'axis',
            formatter: (params: any[]) => {
              let res = `<div style="font-weight: 600; margin-bottom: 4px;">Waktu: ${params[0].axisValue}</div>`;
              params.forEach(item => {
                res += `<div style="display: flex; justify-content: space-between; gap: 20px;">
                  <span>${item.marker} ${item.seriesName}</span>
                  <span style="font-weight: 600;">${item.value} Detik</span>
                </div>`;
              });
              return res;
            }
          },
          legend: { 
            data: ["Rata-rata Latensi AI (detik)"], 
            bottom: 0, 
            textStyle: { color: '#64748b' } 
          },
          grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true },
          xAxis: {
            type: 'category',
            boundaryGap: false,
            data: categories,
            axisLabel: { color: '#64748b' }
          },
          yAxis: {
            type: 'value',
            axisLabel: { 
              color: '#64748b',
              formatter: '{value}s'
            }
          },
          series: [
            {
              name: "Rata-rata Latensi AI (detik)",
              type: 'line',
              smooth: true,
              areaStyle: { opacity: 0.15 },
              itemStyle: { color: '#0ea5e9' },
              data: latencyData
            }
          ]
        };
        lChart.setOption(option);
      }

      const handleResize = () => {
        if (qChart) qChart.resize();
        if (sChart) sChart.resize();
        if (lChart) lChart.resize();
      };
      window.addEventListener('resize', handleResize);

      return () => {
        window.removeEventListener('resize', handleResize);
        if (qChart) qChart.dispose();
        if (sChart) sChart.dispose();
        if (lChart) lChart.dispose();
      };
    }
  }, [activeTab, waChartData, waDateRange, waStartDate, waEndDate]);

  useEffect(() => {
    if (activeTab === "stats" && stats) {
      if (statsTenantsRef.current) {
        const chart = echarts.init(statsTenantsRef.current);
        chart.setOption({
          tooltip: { trigger: 'axis' },
          legend: { show: true, bottom: 0, textStyle: { color: '#64748b' } },
          xAxis: { 
            type: 'category', 
            boundaryGap: false, 
            data: (stats?.tenants || []).map(t => t.date.slice(5)),
            axisLabel: { color: '#64748b' }
          },
          yAxis: { 
            type: 'value',
            axisLabel: { color: '#64748b' }
          },
          series: [{
            name: t("admin.nav_tenants"),
            type: 'line',
            smooth: true,
            areaStyle: { color: 'rgba(45, 212, 191, 0.2)' },
            itemStyle: { color: '#2dd4bf' },
            data: (stats?.tenants || []).map(t => t.count)
          }]
        });
        const handleResize = () => chart.resize();
        window.addEventListener('resize', handleResize);
        return () => { window.removeEventListener('resize', handleResize); chart.dispose(); };
      }
    }
  }, [activeTab, stats]);

  useEffect(() => {
    if (activeTab === "stats" && stats) {
      if (statsUsersRef.current) {
        const chart = echarts.init(statsUsersRef.current);
        chart.setOption({
          tooltip: { trigger: 'axis' },
          legend: { show: true, bottom: 0, textStyle: { color: '#64748b' } },
          xAxis: { 
            type: 'category', 
            boundaryGap: false, 
            data: (stats?.users || []).map(u => u.date.slice(5)),
            axisLabel: { color: '#64748b' }
          },
          yAxis: { 
            type: 'value',
            axisLabel: { color: '#64748b' }
          },
          series: [{
            name: 'Pendaftaran User',
            type: 'line',
            smooth: true,
            areaStyle: { color: 'rgba(251, 191, 36, 0.2)' },
            itemStyle: { color: '#fbbf24' },
            data: (stats?.users || []).map(u => u.count)
          }]
        });
        const handleResize = () => chart.resize();
        window.addEventListener('resize', handleResize);
        return () => { window.removeEventListener('resize', handleResize); chart.dispose(); };
      }
    }
  }, [activeTab, stats]);

  const handleCheckUpdate = async () => {
    setCheckingUpdate(true);
    try {
      const data = await checkUpdate();
      setUpdateInfo(data);
      success("Status update berhasil diperiksa.");
    } catch (err) {
      error(`Gagal memeriksa update: ${err instanceof Error ? err.message : "Error"}`);
    } finally {
      setCheckingUpdate(false);
    }
  };

  const pollUpdateStatus = async () => {
    try {
      const data = await getUpdateStatus();
      setUpdateProgress(data);
      if (data.status === "running") {
        setApplyingUpdate(true);
      } else {
        setApplyingUpdate(false);
      }
      
      // Auto-scroll terminal log to bottom
      if (terminalLogEndRef.current) {
        terminalLogEndRef.current.scrollIntoView({ behavior: "smooth" });
      }
    } catch (err) {
      // Ignore background poll errors
    }
  };

  const handleApplyUpdate = async () => {
    if (!window.confirm("Apakah Anda yakin ingin memulai pembaruan sistem? Ini akan mematikan dan membangun ulang layanan backend & frontend secara otomatis.")) {
      return;
    }
    setApplyingUpdate(true);
    try {
      await applyUpdate();
      success("Proses update sistem dimulai! Silakan pantau log di bawah.");
      pollUpdateStatus();
    } catch (err) {
      error(`Gagal memulai update: ${err instanceof Error ? err.message : "Error"}`);
      setApplyingUpdate(false);
    }
  };

  const loadTenants = async () => {
    setLoading(true);
    try {
      const data = await listTenants();
      setTenants(data);
    } catch (err) {
      error(`Gagal memuat tenant: ${err instanceof Error ? err.message : "Database Error"}`);
    } finally {
      setLoading(false);
    }
  };

  const loadLogs = async () => {
    setLoading(true);
    try {
      const data = await listLogs();
      setLogs(data);
    } catch (err) {
      error(`Gagal memuat log: ${err instanceof Error ? err.message : "Database Error"}`);
    } finally {
      setLoading(false);
    }
  };

  const loadBackups = async () => {
    setLoading(true);
    try {
      const data = await listBackups();
      setBackups(data);
    } catch (err) {
      error(`Gagal memuat list backup: ${err instanceof Error ? err.message : "Network Error"}`);
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async (fromVal?: string, toVal?: string) => {
    try {
      const data = await getGrowthStats(fromVal || dateFrom, toVal || dateTo);
      setStats(data);
    } catch (err) {
      if (err instanceof Error && (err.message.includes("401") || err.message.includes("Unauthorized"))) {
        handleLogout();
        return;
      }
      error(`Gagal memuat statistik: ${err instanceof Error ? err.message : "Unknown error"}`);
    }
  };

  const loadServer = async () => {
    try {
      const data = await getServerStatus();
      setServer(data);
    } catch (err) {
      // Silent refresh
    }
  };

  useEffect(() => {
    if (activeTab === "stats" || activeTab === "dashboard") {
      loadStats(debouncedDateFrom, debouncedDateTo);
    }
  }, [debouncedDateFrom, debouncedDateTo]);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await adminLogin(secret);
      setIsLoggedIn(true);
      success("Autentikasi Berhasil");
    } catch (err) {
      error("Login gagal: " + (err instanceof Error ? err.message : "Secret salah"));
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = () => {
    adminLogout();
    setIsLoggedIn(false);
  };

  const saveGlobalSetting = async (key: string, value: string, isEncrypted: boolean) => {
    await saveAdminGlobalSetting(key, value, isEncrypted);
  };

  const handleSaveNotificationConfig = async (provider: "smtp" | "telegram" | "wa" | "wa_fonnte" | "wa_waha" | "wa_gowa" | "gemini" | "openai" | "claude" | "sumopod" | "active_ai" | "active_wa_bot_ai" | "wa_active_provider" | "wa_bot_system_prompt" | "wa_bot_phone_number") => {
    setSavingConfig(provider);
    try {
      if (provider === "gemini") {
        // Only save API key if user typed a new one (non-empty). Empty = unchanged/masked.
        if (geminiConfig.apiKey.trim()) await saveAdminGlobalSetting("receipt_api_key_gemini", geminiConfig.apiKey, true);
        await saveAdminGlobalSetting("receipt_model_gemini", geminiConfig.model, false);
      } else if (provider === "openai") {
        if (openaiConfig.apiKey.trim()) await saveAdminGlobalSetting("receipt_api_key_openai", openaiConfig.apiKey, true);
        await saveAdminGlobalSetting("receipt_model_openai", openaiConfig.model, false);
      } else if (provider === "claude") {
        if (claudeConfig.apiKey.trim()) await saveAdminGlobalSetting("receipt_api_key_claude", claudeConfig.apiKey, true);
        await saveAdminGlobalSetting("receipt_model_claude", claudeConfig.model, false);
      } else if (provider === "sumopod") {
        if (sumopodConfig.apiKey.trim()) await saveAdminGlobalSetting("receipt_api_key_sumopod", sumopodConfig.apiKey, true);
        await saveAdminGlobalSetting("receipt_model_sumopod", sumopodConfig.model, false);
      } else if (provider === "active_ai") {
        await saveAdminGlobalSetting("receipt_active_ai_provider", activeAI, false);
      } else if (provider === "active_wa_bot_ai") {
        await saveAdminGlobalSetting("wa_bot_active_ai_provider", activeWaBotAI, false);
      } else if (provider === "wa_bot_phone_number") {
        await saveAdminGlobalSetting("wa_bot_phone_number", waBotPhoneNumber, false);
      } else if (provider === "wa_bot_system_prompt") {
        await saveAdminGlobalSetting("wa_bot_system_instructions", waBotSystemPrompt, false);
      } else if (provider === "wa_active_provider") {
        await saveAdminGlobalSetting("notification_wa_active_provider", waActiveProvider, false);
      } else {
        const waKey = provider === "smtp" ? "notification_smtp" : provider === "telegram" ? "notification_telegram" : `notification_${provider}`;
        let config = {};
        if (provider === "smtp") config = smtpConfig;
        else if (provider === "telegram") config = telegramConfig;
        else if (provider === "wa") config = waConfig;
        else if (provider === "wa_fonnte") config = waFonnteConfig;
        else if (provider === "wa_waha") config = waWahaConfig;
        else if (provider === "wa_gowa") config = waGowaConfig;
        
        await saveAdminGlobalSetting(waKey, JSON.stringify(config), true);
      }

      success(`Konfigurasi ${provider.toUpperCase()} berhasil disimpan.`);
    } catch (err) {
      error(`Gagal menyimpan konfigurasi ${provider}: ${err instanceof Error ? err.message : "Internal Error"}`);
    } finally {
      setSavingConfig(null);
    }
  };

  const handleSaveStorageConfig = async (provider: "local" | "s3" | "gdrive" | "active") => {
    setSavingConfig(provider === "active" ? "active_ai" : "smtp" as any); // Reuse saving states or add new ones
    try {
      if (provider === "active") {
        await saveAdminGlobalSetting("storage_active_provider", storageActiveProvider, false);
      } else if (provider === "s3") {
        const toSave = { ...s3Config };
        if (!toSave.secretKey) {
          // If empty, don't overwrite the existing saved secret
          const existing = await getGlobalSetting("storage_s3_config");
          if (existing && existing.value) {
            const parsed = JSON.parse(existing.value);
            toSave.secretKey = parsed.secretKey;
          }
        }
        await saveAdminGlobalSetting("storage_s3_config", JSON.stringify(toSave), true);
      } else if (provider === "gdrive") {
        const toSave = { ...gdriveConfig };
        if (!toSave.credentialsJson) {
          const existing = await getGlobalSetting("storage_gdrive_config");
          if (existing && existing.value) {
            const parsed = JSON.parse(existing.value);
            toSave.credentialsJson = parsed.credentialsJson;
          }
        }
        await saveAdminGlobalSetting("storage_gdrive_config", JSON.stringify(toSave), true);
      } else if (provider === "local") {
        await saveAdminGlobalSetting("storage_local_config", JSON.stringify(localConfig), false);
      }
      success(`Konfigurasi Storage ${provider.toUpperCase()} berhasil disimpan.`);
    } catch (err) {
      error(`Gagal menyimpan konfigurasi storage: ${err instanceof Error ? err.message : "Internal Error"}`);
    } finally {
      setSavingConfig(null);
    }
  };

  const handleSaveDbConfig = async () => {
    setSavingConfig("database");
    try {
      await saveAdminGlobalSetting("database_config", JSON.stringify(dbConfig), true);
      success("Konfigurasi Database berhasil disimpan.");
    } catch (err) {
      error("Terjadi kesalahan saat test notifikasi.");
    } finally {
      setTestingConfig(null);
    }
  };

  const handleCreateBackup = async () => {
    setIsBackingUp(true);
    try {
      await createBackup(backupType);
      success("Backup berhasil dibuat.");
      loadBackups();
    } catch (err) {
      error("Gagal membuat backup.");
    } finally {
      setIsBackingUp(false);
    }
  };

  const handleRestoreBackup = async () => {
    if (!fileToRestore) return;
    setIsRestoring(true);
    try {
      await restoreBackup(fileToRestore);
      success("Database berhasil direstore.");
      setFileToRestore(null);
    } catch (err) {
      error("Gagal melakukan restore: " + (err instanceof Error ? err.message : "Internal Error"));
    } finally {
      setIsRestoring(false);
    }
  };

  const handleDownloadBackup = async (filename: string) => {
    try {
      const blob = await downloadBackupBlob(filename);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      error("Gagal mengunduh backup.");
    }
  };

  const handleDownloadTenantBackup = async (filename: string) => {
    if (!editingTenant) return;
    try {
      const blob = await downloadTenantBackupBlob(editingTenant.id, filename);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (err) {
      error("Gagal mengunduh backup tenant.");
    }
  };

  const handleSaveOptimization = async () => {
    setSavingConfig("active_ai");
    try {
      await saveAdminGlobalSetting("optimization_config", JSON.stringify(optConfig), false);
      success("Pengaturan Optimasi berhasil disimpan.");
    } catch (err) {
      error("Gagal menyimpan optimasi: " + (err instanceof Error ? err.message : "Error"));
    } finally {
      setSavingConfig(null);
    }
  };

  const handleExecuteQuery = async () => {
    if (!sqlQuery.trim()) return;
    setIsExecutingQuery(true);
    setQueryResult(null);
    try {
      const res = await executeQuery(sqlQuery);
      setQueryResult(res);
      success("Query berhasil dieksekusi.");
    } catch (err) {
      setQueryResult({ columns: [], rows: [], error: err instanceof Error ? err.message : "Database Error" });
      error("Gagal eksekusi query.");
    } finally {
      setIsExecutingQuery(false);
    }
  };
  
  const maskValue = (key: string, value: any): string => {
    if (value === null || value === undefined || value === "NULL") return "NULL";
    if (!isMasked) return String(value);
    
    const str = String(value);
    const lowerKey = key.toLowerCase();
    
    // Mask UUIDs and similar IDs
    if (lowerKey.includes("_id") || lowerKey === "id" || lowerKey.includes("_by")) {
      if (str.length > 10) {
        return `${str.slice(0, 4)}...${str.slice(-4)}`;
      }
    }
    
    // Mask email patterns
    if (lowerKey.includes("email")) {
      const parts = str.split("@");
      if (parts.length === 2) {
        return `${parts[0].slice(0, 2)}***@${parts[1]}`;
      }
    }
    
    // Mask phone numbers
    if (lowerKey.includes("phone")) {
      return str.slice(0, 3) + "****" + str.slice(-2);
    }
    
    // Mask tokens or keys
    if (lowerKey.includes("token") || lowerKey.includes("secret") || lowerKey.includes("key")) {
      return "********";
    }
    
    return str;
  };

  const loadDbStats = async () => {
    try {
      const data = await getDatabaseStats();
      setDbStats(data);
    } catch (err: any) {
      error("Gagal memuat statistik database: " + err.message);
    }
  };

  const loadDbGrowth = async () => {
    try {
      const data = await getDatabaseGrowth();
      setDbGrowth(data);
    } catch (err: any) {
      error("Gagal memuat data pertumbuhan database: " + err.message);
    }
  };

  const loadWhatsAppQueueChartData = async () => {
    try {
      const data = await getWhatsAppQueueHistory(100, 0, "");
      setWaChartData(data.items || []);
    } catch (err: any) {
      console.error("Gagal memuat data chart WhatsApp:", err);
    }
  };

  const loadWhatsAppQueueStats = async () => {
    try {
      const data = await getWhatsAppQueueStats();
      setWaQueueStats(data);
      loadWhatsAppQueueChartData();
    } catch (err: any) {
      error("Gagal memuat statistik antrean WhatsApp: " + err.message);
    }
  };

  const loadWhatsAppQueueHistory = async (limitVal = waQueueLimit, offsetVal = waQueueOffset, searchVal = waQueueSearch) => {
    setLoadingWaQueue(true);
    try {
      const data = await getWhatsAppQueueHistory(limitVal, offsetVal, searchVal);
      setWaQueueHistory(data.items || []);
      setWaQueueTotal(data.total || 0);
    } catch (err: any) {
      error("Gagal memuat riwayat antrean WhatsApp: " + err.message);
    } finally {
      setLoadingWaQueue(false);
    }
  };

  const handleRetryWhatsAppMessage = async (id: string) => {
    setRetryingMsgId(id);
    try {
      await retryWhatsAppQueueMessage(id);
      success("Pesan berhasil dimasukkan kembali ke antrean!");
      loadWhatsAppQueueStats();
      loadWhatsAppQueueHistory();
    } catch (err: any) {
      error("Gagal mengulang pesan: " + err.message);
    } finally {
      setRetryingMsgId(null);
    }
  };

  useEffect(() => {
    if (isLoggedIn && activeTab === "whatsapp") {
      loadWhatsAppQueueHistory(waQueueLimit, waQueueOffset, debouncedWaSearch);
    }
  }, [isLoggedIn, activeTab, waQueueLimit, waQueueOffset, debouncedWaSearch]);

  const handleTestDbConfig = async () => {
    setTestingConfig("database");
    try {
      await testDatabase(JSON.stringify(dbConfig));
      success("Koneksi Database berhasil!");
    } catch (err) {
      error("Gagal terhubung ke Database: " + (err instanceof Error ? err.message : "Error"));
    } finally {
      setTestingConfig(null);
    }
  };

  const handleTestConnection = async (provider: string) => {
    if (provider === "gemini" || provider === "openai" || provider === "claude" || provider === "sumopod") {
      setTestingConfig(provider);
      try {
        let key = "";
        if (provider === "gemini") key = geminiConfig.apiKey;
        else if (provider === "openai") key = openaiConfig.apiKey;
        else if (provider === "claude") key = claudeConfig.apiKey;
        else if (provider === "sumopod") key = sumopodConfig.apiKey;

        const res: any = await testAI(provider, key);
        console.log(`[Admin] Fetch models for ${provider} success:`, res.models);
        if (res.models && res.models.length > 0) {
          if (provider === "gemini") {
            setAiModels(res.models);
            localStorage.setItem("admin_ai_models", JSON.stringify(res.models));
          } else if (provider === "openai") {
            setOpenaiModels(res.models);
            localStorage.setItem("admin_openai_models", JSON.stringify(res.models));
          } else if (provider === "claude") {
            setClaudeModels(res.models);
            localStorage.setItem("admin_claude_models", JSON.stringify(res.models));
          } else if (provider === "sumopod") {
            setSumopodModels(res.models);
            localStorage.setItem("admin_sumopod_models", JSON.stringify(res.models));
          }
        }
        success(`Koneksi ${provider.toUpperCase()} berhasil! Daftar model diperbarui.`);
      } catch (e: any) {
        error(`Gagal menghubungkan ke ${provider}: ` + (e.message || "Unknown error"));
      } finally {
        setTestingConfig(null);
      }
      return;
    }

    setTestProvider(provider);
    setTestDestination("");
    setShowTestModal(true);
  };

  const confirmTestConnection = async () => {
    if (!testDestination) {
      error("Tujuan pengiriman tes tidak boleh kosong");
      return;
    }
    
    setShowTestModal(false);
    setTestingConfig(testProvider);
    try {
      let config = {};
      if (testProvider === "smtp") config = smtpConfig;
      else if (testProvider === "telegram") config = telegramConfig;
      else if (testProvider === "wa") config = waConfig;
      else if (testProvider === "wa_fonnte") config = waFonnteConfig;
      else if (testProvider === "wa_waha") config = waWahaConfig;
      else if (testProvider === "wa_gowa") config = waGowaConfig;

      await testNotification(testProvider, JSON.stringify(config), testDestination);
      success(`Test koneksi ${testProvider.toUpperCase()} berhasil. Pesan uji coba terkirim ke ${testDestination}.`);
    } catch (err) {
      error(`Test koneksi ${testProvider} gagal: ${err instanceof Error ? err.message : "Periksa konfigurasi Anda"}`);
    } finally {
      setTestingConfig(null);
    }
  };

  const generateTenantCode = () => {
    const code = "T" + Math.floor(1000 + Math.random() * 9000);
    setForm(prev => ({ ...prev, tenant_code: code }));
  };

  const handleUpdateQuota = async (tenantID: string, users: number, transactions: number) => {
    try {
      await updateTenantQuotas(tenantID, users, transactions);
      success("Kuota diperbarui");
      loadTenants();
    } catch (err) {
      error("Gagal memperbarui kuota");
    }
  };

  const handleManageModules = async (tenantID: string) => {
    setSelectedTenantID(tenantID);
    try {
      const data = await listTenantModules(tenantID);
      setSelectedModules(data);
    } catch (err) {
      error("Gagal memuat modul");
    }
  };

  const handleToggleModule = async (moduleCode: string, isEnabled: boolean) => {
    if (!selectedTenantID) return;
    try {
      await updateTenantModule(selectedTenantID, moduleCode, isEnabled);
      success(`Modul ${moduleCode.toUpperCase()} diperbarui`);
      const data = await listTenantModules(selectedTenantID);
      setSelectedModules(data);
    } catch (err) {
      error("Gagal memperbarui modul");
    }
  };

  const handleImpersonate = async (tenantID: string, code: string) => {
    const email = `admin@${code}.local`; 
    try {
      // Use a valid UUID format for the placeholder ID to avoid DB syntax errors
      const res = await impersonateUser("00000000-0000-0000-0000-000000000000", tenantID, email);
      // res.access_token is still returned but no longer stored in localStorage (handled by cookies)
      success(`Beralih ke akun ${email}...`);
      setTimeout(() => window.location.href = `/app/${code}/finance/dashboard`, 1200);
    } catch (err) {
      error("Gagal melakukan impersonasi");
    }
  };

  const handleDeleteTenant = async (id: string) => {
    setLoading(true);
    try {
      await deleteTenant(id);
      success("Workspace berhasil dihapus");
      setTenantToDelete(null);
      loadTenants();
      loadStats(debouncedDateFrom, debouncedDateTo);
    } catch (err) {
      error("Gagal menghapus tenant: " + (err instanceof Error ? err.message : "Database Error"));
    } finally {
      setLoading(false);
    }
  };

  const handleUserReset = async () => {
    if (!userToReset || !resetValue || !editingTenant) return;
    setIsResetting(true);
    try {
      const { user, action } = userToReset;
      if (action === 'password') {
        await adminResetUserPassword(user.id, resetValue);
        success(`Password untuk ${user.full_name} berhasil direset.`);
      } else if (action === 'email') {
        await adminUpdateUserEmail(user.id, resetValue);
        success(`Email untuk ${user.full_name} berhasil diubah.`);
      } else if (action === 'phone') {
        await adminUpdateUserPhone(user.id, resetValue);
        success(`Nomor HP untuk ${user.full_name} berhasil diubah.`);
      }
      
      // Close modal first so user knows it's done
      const currentTenantId = editingTenant.id;
      setUserToReset(null);
      setResetValue("");

      // Try refresh list separately
      try {
        const users = await listTenantUsers(currentTenantId);
        setTenantUsers(users);
      } catch (refreshErr) {
        console.error("Refresh list failed after update:", refreshErr);
      }
    } catch (err) {
      error("Gagal melakukan update user. Silakan coba lagi.");
    } finally {
      setIsResetting(false);
    }
  };

  const handleCreateTenantBackup = async () => {
    if (!editingTenant) return;
    setLoading(true);
    try {
      await createTenantBackup(editingTenant.id);
      success("Backup tenant berhasil dibuat.");
      const backups = await listTenantBackups(editingTenant.id);
      setTenantBackups(backups);
    } catch (err) {
      error("Gagal membuat backup tenant.");
    } finally {
      setLoading(false);
    }
  };

  const handleRestoreTenantBackup = async (filename: string) => {
    if (!editingTenant || !window.confirm(`Restore tenant "${editingTenant.name}" ke backup "${filename}"? Tindakan ini akan menimpa data saat ini.`)) return;
    setIsRestoring(true);
    try {
      await restoreTenantBackup(editingTenant.id, filename);
      success("Restore tenant berhasil diselesaikan.");
    } catch (err) {
      error("Gagal melakukan restore tenant.");
    } finally {
      setIsRestoring(false);
    }
  };

  const handleUpdateTenant = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingTenant) return;
    setLoading(true);
    try {
      await updateTenant(editingTenant.id, editingTenant.name, editingTenant.status);
      success("Workspace berhasil diperbarui");
      setEditingTenant(null);
      loadTenants();
    } catch (err) {
      error("Gagal memperbarui workspace");
    } finally {
      setLoading(false);
    }
  };

  const handleSubmitBootstrap = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await bootstrapTenant({
        ...form,
        tenant_code: form.tenant_code.trim(),
        admin_email: form.admin_email.trim()
      });
      success(t("admin.tenant_created_success"));
      setForm({
        tenant_code: "",
        tenant_name: "",
        admin_email: "owner@pekan.local",
        admin_name: "Workspace Owner",
        password: "password"
      });
      loadTenants();
      loadStats();
      setActiveTab("tenants");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Error";
      if (msg.includes("tenant already exists")) {
        error("Kode Workspace sudah digunakan. Silakan gunakan kode lain.");
      } else if (msg.includes("user already exists")) {
        error("Email Admin sudah terdaftar. Silakan gunakan email lain.");
      } else {
        error(`Gagal membuat workspace: ${msg}`);
      }
    } finally {
      setLoading(false);
    }
  };

  if (!isLoggedIn) {
    return (
      <section className="auth-wrap">
        <div className="auth-card">
          <p className="auth-kicker">Platform Admin</p>
          <h1 className="auth-title">Admin Central</h1>
          <p className="page-subtitle">Akses Manajemen Infrastruktur</p>
          <form className="form-grid spacing-mt-lg" onSubmit={handleLogin}>
            <div className="form-field">
              <input type="text" name="username" value="admin" style={{ display: "none" }} autoComplete="username" readOnly />
              <label>Admin Secret Key</label>
              <input 
                className="input-control" 
                type="password" 
                value={secret} 
                onChange={(e) => setSecret(e.target.value)} 
                placeholder="Masukkan secret"
                required 
                autoComplete="current-password"
              />
            </div>
            <button className="btn btn-primary" type="submit" disabled={loading} style={{ width: "100%", height: "46px" }}>
              {loading ? "Memverifikasi..." : "Akses Dashboard"}
            </button>
          </form>
          <p style={{ marginTop: "2rem", fontSize: "0.8rem", color: "var(--muted)", textAlign: "center" }}>
            Periksa <code>JWT_SECRET</code> pada environment server.
          </p>
        </div>
        <ToastContainer toasts={toasts} onRemove={remove} />
      <BackToTop />
      </section>
    );
  }

  const renderKPIs = () => {
    if (!stats) return null;
    
    // Dynamic health status
    const isDbHealthy = server?.db_status?.toLowerCase().includes("healthy") || server?.db_status?.toLowerCase().includes("online");
    const isRedisHealthy = server?.redis_status?.toLowerCase().includes("healthy") || server?.redis_status?.toLowerCase().includes("running");
    
    let healthLabel = "Normal";
    let healthColor = "#22c55e"; // Green

    if (!server) {
      healthLabel = "Loading...";
      healthColor = "#94a3b8"; // Gray
    } else if (!isDbHealthy || !isRedisHealthy) {
      healthLabel = "Down";
      healthColor = "#ef4444"; // Red
    } else if (server?.db_status?.toLowerCase().includes("degraded")) {
      healthLabel = "Degraded";
      healthColor = "#eab308"; // Yellow
    }

    return (
      <div className="card-grid" style={{ gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))", marginBottom: "2rem" }}>
        <div className="surface card shadow-soft stat-card">
          <p className="stat-label">{t("admin.nav_tenants")}</p>
          <h2 className="stat-value">{stats.total_tenants}</h2>
          <p className="stat-meta">{t("admin.server.realtime")}</p>
        </div>
        <div className="surface card shadow-soft stat-card">
          <p className="stat-label">{t("common.user")}</p>
          <h2 className="stat-value">{stats.total_users}</h2>
          <p className="stat-meta">Akun terdaftar</p>
        </div>
        <div className="surface card shadow-soft stat-card">
          <p className="stat-label">{t("admin.kpi_system_health")}</p>
          <h2 className="stat-value" style={{ color: healthColor }}>{healthLabel}</h2>
          <p className="stat-meta">{server?.db_status || t("admin.system_status_ok")}</p>
        </div>
      </div>
    );
  };

  return (
    <div className={`app-shell admin-layout ${sidebarOpen ? "sidebar-open" : ""}`}>
      <aside className="app-sidebar">
        <div className="sidebar-header">
          <div>
            <h2 className="brand">{t("app.brand")}</h2>
            <p className="sidebar-caption">{t("admin.brand_tag")}</p>
          </div>
          <button type="button" className="btn btn-ghost-inline sidebar-close-btn" onClick={() => setSidebarOpen(false)} aria-label={t("common.closeMenu")}>
            <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
        <nav className="app-nav">
          <button className={`app-nav-link ${activeTab === "dashboard" ? "is-active" : ""}`} onClick={() => { setActiveTab("dashboard"); setSidebarOpen(false); }}>{t("admin.nav_dashboard")}</button>
          
          {/* Tenant Management Group */}
          <button 
            type="button" 
            className={`app-nav-link sidebar-expand-btn ${(activeTab === "tenants" || activeTab === "add_tenant") ? "is-expanded" : ""}`}
            onClick={() => setTenantsExpanded(!tenantsExpanded)}
            style={{ width: "100%", textAlign: "left", display: "flex", alignItems: "center", justifyContent: "space-between" }}
          >
            <span>{t("admin.nav_tenant_mgmt")}</span>
            <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor" style={{ transform: tenantsExpanded ? "rotate(180deg)" : "none", transition: "transform 0.2s", opacity: 0.7 }}>
              <path d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z" />
            </svg>
          </button>
          
          {tenantsExpanded && (
            <div className="sidebar-sub-nav">
              <button className={`app-nav-link sub-link ${activeTab === "tenants" ? "is-active" : ""}`} onClick={() => { setActiveTab("tenants"); setSidebarOpen(false); }}>{t("admin.nav_tenants")}</button>
              <button className={`app-nav-link sub-link ${activeTab === "add_tenant" ? "is-active" : ""}`} onClick={() => { setActiveTab("add_tenant"); setSidebarOpen(false); }}>Tambah Workspace</button>
            </div>
          )}

          <button className={`app-nav-link ${activeTab === "stats" ? "is-active" : ""}`} onClick={() => { setActiveTab("stats"); setSidebarOpen(false); }}>{t("admin.nav_stats")}</button>
          
          {/* System Group */}
          <button 
            type="button" 
            className={`app-nav-link sidebar-expand-btn ${(activeTab === "server" || activeTab === "logs") ? "is-expanded" : ""}`}
            onClick={() => setSystemExpanded(!systemExpanded)}
            style={{ width: "100%", textAlign: "left", display: "flex", alignItems: "center", justifyContent: "space-between" }}
          >
            <span>{t("admin.nav_system_infra")}</span>
            <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor" style={{ transform: systemExpanded ? "rotate(180deg)" : "none", transition: "transform 0.2s", opacity: 0.7 }}>
              <path d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z" />
            </svg>
          </button>

          {systemExpanded && (
            <div className="sidebar-sub-nav">
              <button className={`app-nav-link sub-link ${activeTab === "server" ? "is-active" : ""}`} onClick={() => { setActiveTab("server"); setSidebarOpen(false); }}>{t("admin.nav_server")}</button>
              <button className={`app-nav-link sub-link ${activeTab === "logs" ? "is-active" : ""}`} onClick={() => { setActiveTab("logs"); setSidebarOpen(false); }}>{t("admin.nav_logs")}</button>
              <button className={`app-nav-link sub-link ${activeTab === "backups" ? "is-active" : ""}`} onClick={() => { setActiveTab("backups"); setSidebarOpen(false); }}>Backup & Restore</button>
              <button className={`app-nav-link sub-link ${activeTab === "updates" ? "is-active" : ""}`} onClick={() => { setActiveTab("updates"); setSidebarOpen(false); }}>System Update</button>
              <button className={`app-nav-link sub-link ${activeTab === "ai" ? "is-active" : ""}`} onClick={() => { setActiveTab("ai"); setSidebarOpen(false); }}>AI Settings</button>
              <button className={`app-nav-link sub-link ${activeTab === "whatsapp" ? "is-active" : ""}`} onClick={() => { setActiveTab("whatsapp"); setSidebarOpen(false); }}>Chat AI Queue</button>
              <button className={`app-nav-link sub-link ${activeTab === "notifications" ? "is-active" : ""}`} onClick={() => { setActiveTab("notifications"); setSidebarOpen(false); }}>Notification Providers</button>
              <button className={`app-nav-link sub-link ${activeTab === "database" ? "is-active" : ""}`} onClick={() => { setActiveTab("database"); setSidebarOpen(false); }}>Database Config</button>
              <button className={`app-nav-link sub-link ${activeTab === "storage" ? "is-active" : ""}`} onClick={() => { setActiveTab("storage"); setSidebarOpen(false); }}>Storage & Cloud</button>
              <button className={`app-nav-link sub-link ${activeTab === "optimization" ? "is-active" : ""}`} onClick={() => { setActiveTab("optimization"); setSidebarOpen(false); }}>Optimasi & Performa</button>
              <button className={`app-nav-link sub-link ${activeTab === "dbtool" ? "is-active" : ""}`} onClick={() => { setActiveTab("dbtool"); setSidebarOpen(false); }}>Database Size & Growth</button>
            </div>
          )}
        </nav>
        <div className="sidebar-footer">
          <div className="sidebar-user" onClick={() => setProfileMenuOpen(!profileMenuOpen)} style={{ cursor: "pointer" }}>
            <div className="sidebar-user-avatar">
              <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                <path d="M12 12a4 4 0 1 0-4-4 4 4 0 0 0 4 4Zm0 2c-4 0-7 2-7 4.5V20h14v-1.5C19 16 16 14 12 14Z" />
              </svg>
            </div>
            <div className="sidebar-user-info">
              <span className="sidebar-user-name">Administrator</span>
              <span className="sidebar-user-role">System Root</span>
            </div>
            <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" style={{ transform: profileMenuOpen ? "rotate(180deg)" : "none", transition: "transform 0.2s" }}>
              <path d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z" />
            </svg>
          </div>

          {profileMenuOpen && (
            <div className="sidebar-profile-menu">
              <button className="app-nav-link sub-link danger" onClick={handleLogout} style={{ background: "none", border: "none", width: "100%", textAlign: "left", cursor: "pointer", color: "#ff8a8a" }}>
                {t("nav.logout")}
              </button>
            </div>
          )}

          <div className="sidebar-footer-actions">
            <button className="sidebar-footer-btn" onClick={() => setLocale(locale === "id" ? "en" : "id")}>
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" style={{ marginRight: "6px" }}>
                <circle cx="12" cy="12" r="10" />
                <line x1="2" y1="12" x2="22" y2="12" />
                <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
              </svg>
              {locale.toUpperCase()}
            </button>
          </div>
        </div>
      </aside>

      <button type="button" className={`sidebar-backdrop${sidebarOpen ? " is-visible" : ""}`} onClick={() => setSidebarOpen(false)} aria-label={t("common.closeSidebar")} />

      <header className="mobile-header surface">
        <button type="button" className="btn btn-ghost-inline sidebar-toggle" onClick={() => setSidebarOpen(true)} aria-label={t("common.toggleMenu")}>
          <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
            <path d="M3 6h18v2H3V6Zm0 5h18v2H3v-2Zm0 5h18v2H3v-2Z" />
          </svg>
        </button>
        <strong className="mobile-brand">{t("admin.brand_tag")}</strong>
      </header>

      <div className="app-main">

        <main className="app-content">
          <PageHeader 
            title={
              activeTab === "tenants" ? t("admin.tenants.title") : 
              activeTab === "add_tenant" ? t("admin.tenants.add") : 
              activeTab === "logs" ? t("admin.logs.title") : 
              activeTab === "stats" ? t("admin.nav_stats") : 
              activeTab === "server" ? t("admin.nav_server") : 
              activeTab === "notifications" ? t("admin.notifications.title") :
              activeTab === "database" ? "Database Configuration" :
              activeTab === "backups" ? "Backup & Restore" :
              activeTab === "whatsapp" ? "Chat AI Queue & Statistics" :
              activeTab === "updates" ? "System Auto-Updater" :
              t("admin.nav_dashboard")
            }
            description={
              activeTab === "tenants" ? t("admin.tenants.subtitle") :
              activeTab === "add_tenant" ? "Tambahkan workspace baru dengan kuota yang disesuaikan." :
              activeTab === "logs" ? "Pantau log aktivitas audit di seluruh platform." :
              activeTab === "stats" ? "Analisis pertumbuhan workspace dan penggunaan sistem." :
              activeTab === "server" ? "Informasi status dan kesehatan server infrastruktur." :
              activeTab === "notifications" ? t("admin.notifications.subtitle") :
              activeTab === "database" ? "Kelola pengaturan koneksi PostgreSQL database Anda." :
              activeTab === "backups" ? "Kelola file dump database untuk keamanan data dan migrasi." :
              activeTab === "whatsapp" ? "Pantau statistik antrean pesan real-time, status pemrosesan AI, log error, dan retry manual." :
              activeTab === "updates" ? "Periksa, unduh, dan pasang pembaruan kode aplikasi Pekan langsung dari GitHub secara aman." :
              "Pusat kendali dan infrastruktur Pekan"
            }
            hideInfo={true}
          />


          {activeTab === "dashboard" && (
            <div className="dashboard-view">
              {renderKPIs()}
              
              {stats && (
                <div className="card-grid two-col">
                  <div className="surface card shadow-soft">
                    <h3 className="form-title">Riwayat Pertumbuhan Platform</h3>
                    <div ref={chartRef} style={{ height: "300px", width: "100%" }} />
                  </div>
                  <div className="surface card shadow-soft">
                    <h3 className="form-title">Ringkasan Infrastruktur</h3>
                    <div className="info-grid">
                      <div className="info-item">
                        <span className="info-label">Hostname</span>
                        <strong className="info-value">{server?.ip_address || "-"}</strong>
                      </div>
                      <div className="info-item">
                        <span className="info-label">OS</span>
                        <strong className="info-value">{server?.os || "-"}</strong>
                      </div>
                      <div className="info-item">
                        <span className="info-label">Database {t("settings.roles.status")}</span>
                        <span className={`info-value ${server?.db_status === "Online" ? "status-online" : "status-offline"}`}>
                          ● {server?.db_status || "Checking..."}
                        </span>
                      </div>
                      <div className="info-item">
                        <span className="info-label">Redis Status</span>
                        <span className={`info-value ${server?.redis_status === "Online" ? "status-online" : "status-offline"}`}>
                          ● {server?.redis_status || "Checking..."}
                        </span>
                      </div>
                      <div className="info-item">
                        <span className="info-label">Uptime</span>
                        <strong className="info-value">{server?.uptime || "-"}</strong>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}

          {activeTab === "add_tenant" && (
            <div className="card-grid tight">
              <div className="surface card shadow-soft" style={{ gridColumn: "1 / -1" }}>
                <div className="form-header">
                  <h2 className="form-section-title">{t("admin.boot.title")}</h2>
                  <p className="form-section-desc">{t("admin.boot.subtitle")}</p>
                </div>
                <form className="form-grid" onSubmit={handleSubmitBootstrap}>
                  <div className="card-grid two-col" style={{ gap: "1.25rem", marginBottom: "1rem", alignItems: "start" }}>
                    <div className="form-field">
                      <label>Kode Workspace</label>
                      <div style={{ display: "flex", gap: "8px", width: "100%" }}>
                        <input 
                          className="input-control" 
                          value={form.tenant_code} 
                          onChange={(e) => setForm({ ...form, tenant_code: e.target.value })} 
                          placeholder="ACME" 
                          required 
                          style={{ flex: "1 1 auto", minWidth: "100px" }} 
                        />
                        <button 
                          type="button" 
                          className="btn btn-secondary-outline" 
                          onClick={generateTenantCode} 
                          title="Auto-generate" 
                          style={{ width: "auto", padding: "0 15px", flexShrink: 0, height: "42px" }}
                        >
                          <svg viewBox="0 0 24 24" width="18" height="18"><path d="M17.65 6.35A7.958 7.958 0 0 0 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0 1 12 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35Z" fill="currentColor"/></svg>
                        </button>
                      </div>
                    </div>
                    <div className="form-field">
                      <label>{t("admin.tenants.table.workspace")}</label>
                      <input className="input-control" value={form.tenant_name} onChange={(e) => setForm({ ...form, tenant_name: e.target.value })} placeholder="Workspace Utama" required />
                    </div>
                  </div>
                  <div className="card-grid two-col" style={{ gap: "1.25rem", marginBottom: "1rem", alignItems: "start" }}>
                    <div className="form-field">
                      <label>Email Administrator</label>
                      <input className="input-control" type="email" value={form.admin_email} onChange={(e) => setForm({ ...form, admin_email: e.target.value })} required style={{ width: "100%" }} autoComplete="username" />
                    </div>
                    <div className="form-field">
                      <label style={{ display: 'block', marginBottom: '0.35rem' }}>Password</label>
                      <PasswordInput value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} required autoComplete="new-password" />
                      <div style={{ marginTop: "1rem" }}>
                        <PasswordStrength password={form.password} />
                      </div>
                    </div>
                  </div>
                  <div className="form-actions-inline" style={{ marginTop: "1.5rem", display: "flex", gap: "10px" }}>
                    <button className="btn btn-primary" type="submit" disabled={loading}>
                      {loading ? t("common.loading") : t("admin.boot.submit")}
                    </button>
                    <button className="btn btn-secondary-outline" type="button" onClick={() => setActiveTab("tenants")}>{t("admin.boot.cancel")}</button>
                  </div>
                </form>
              </div>
            </div>
          )}

          {activeTab === "tenants" && (
            <>
              <div className="card-grid">
                <div className="surface card shadow-soft">
                  <div className="card-header-actions">
                    <h3 className="form-title">{t("admin.tenants.title")}</h3>
                    <div className="header-controls">
                      <div className="search-box">
                         <input 
                           type="text" 
                           className="input-control has-icon" 
                           placeholder={t("admin.tenants.search")} 
                           value={searchTerm}
                           onChange={(e) => setSearchTerm(e.target.value)}
                         />
                         <svg className="search-icon" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
                      </div>
                      <button className="btn btn-primary" onClick={() => setActiveTab("add_tenant")}>{t("admin.tenants.add")}</button>
                    </div>
                  </div>
                  <div className="data-table-wrap table-mobile-stack">
                    <table className="data-table">
                      <thead>
                        <tr>
                          <th>{t("admin.tenants.table.workspace")}</th>
                          <th>{t("admin.tenants.table.quota")}</th>
                          <th>{t("admin.tenants.table.status")}</th>
                          <th className="text-right">{t("admin.tenants.table.action")}</th>
                        </tr>
                      </thead>
                      <tbody>
                        {(tenants || []).filter(tenant => (tenant.name || "").toLowerCase().includes(debouncedSearch.toLowerCase()) || (tenant.code || "").toLowerCase().includes(debouncedSearch.toLowerCase())).length === 0 && !loading && <tr><td colSpan={4} className="text-center-muted">{t("common.noData")}</td></tr>}
                        {(tenants || []).filter(tenant => (tenant.name || "").toLowerCase().includes(debouncedSearch.toLowerCase()) || (tenant.code || "").toLowerCase().includes(debouncedSearch.toLowerCase()))
                          .map(tenant => (
                          <tr key={tenant.id}>
                            <td data-label={t("admin.tenants.table.workspace")}>
                              <strong>{tenant.name}</strong><br/>
                              <code className="text-xs">{tenant.code}</code>
                            </td>
                            <td data-label={t("admin.tenants.table.quota")}>
                              <div className="quota-controls">
                                <input type="number" className="input-control quota-input-sm" defaultValue={tenant.quota_users} onBlur={(e) => handleUpdateQuota(tenant.id, parseInt(e.target.value), tenant.quota_transactions)} />
                                <span>/</span>
                                <input type="number" className="input-control quota-input-md" defaultValue={tenant.quota_transactions} onBlur={(e) => handleUpdateQuota(tenant.id, tenant.quota_users, parseInt(e.target.value))} />
                              </div>
                            </td>
                            <td data-label={t("admin.tenants.table.status")}><span className="badge-status running">{tenant.user_count} {t("settings.users.statusActive")}</span></td>
                            <td data-label={t("admin.tenants.table.action")} className="text-right">
                              <div className="table-actions">
                                <button className="btn btn-ghost-inline btn-sm" onClick={() => setEditingTenant(tenant)}>{t("common.edit")}</button>
                                <button 
                                  className="btn btn-ghost-inline btn-sm" 
                                  onClick={() => {
                                    setEditingTenant(tenant);
                                    setTenantModalTab('backups');
                                  }}
                                >
                                  Backups
                                </button>
                                <button className="btn btn-ghost-inline btn-sm" onClick={() => handleImpersonate(tenant.id, tenant.code)}>Impersonate</button>
                                <button className="btn btn-ghost-inline btn-sm danger" onClick={() => setTenantToDelete(tenant)}>{t("common.delete")}</button>
                              </div>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
                
                {selectedTenantID && (
                   <div className="surface card shadow-soft spacing-mt-lg">
                      <h3 className="form-title">Modul Fitur: {tenants.find(t => t.id === selectedTenantID)?.name}</h3>
                      <div className="badge-grid">
                        {["finance", "inventory", "hrm", "crm"].map(code => {
                          const m = (selectedModules || []).find(sm => sm.module_code === code);
                          const isEnabled = m?.is_enabled ?? false;
                          return (
                            <label key={code} className={`badge-selector ${isEnabled ? "is-active" : ""}`}>
                              <input type="checkbox" checked={isEnabled} onChange={(e) => handleToggleModule(code, e.target.checked)} />
                              <span>{code}</span>
                            </label>
                          );
                        })}
                      </div>
                    </div>
                  )}
                </div>

                {editingTenant && (
                  <div className="modal-overlay">
                    <div className="surface card shadow-strong modal-md" style={{ display: "flex", flexDirection: "column", maxHeight: "90vh" }}>
                      <div style={{ paddingBottom: "1rem", borderBottom: "1px solid var(--border)", marginBottom: "1rem" }}>
                        <h3 className="form-title" style={{ margin: 0 }}>{t("admin.tenants.edit")}: {editingTenant.code}</h3>
                        <div style={{ display: "flex", gap: "1rem", marginTop: "1rem" }}>
                          <button 
                            className={`btn ${tenantModalTab === 'info' ? 'btn-primary' : 'btn-ghost-inline'}`} 
                            style={{ fontSize: "0.85rem", padding: "6px 15px", minHeight: "auto" }}
                            onClick={() => setTenantModalTab('info')}
                          >
                            Info & Users
                          </button>
                          <button 
                            className={`btn ${tenantModalTab === 'backups' ? 'btn-primary' : 'btn-ghost-inline'}`} 
                            style={{ fontSize: "0.85rem", padding: "6px 15px", minHeight: "auto" }}
                            onClick={() => setTenantModalTab('backups')}
                          >
                            Backup & Restore
                          </button>
                          <button 
                            className={`btn ${tenantModalTab === 'modules' ? 'btn-primary' : 'btn-ghost-inline'}`} 
                            style={{ fontSize: "0.85rem", padding: "6px 15px", minHeight: "auto" }}
                            onClick={() => {
                              setTenantModalTab('modules');
                              handleManageModules(editingTenant.id);
                            }}
                          >
                            Modules
                          </button>
                        </div>
                      </div>

                      <div style={{ flex: 1, overflowY: "auto", paddingRight: "5px" }}>
                        {tenantModalTab === 'info' ? (
                          <form className="form-grid" onSubmit={handleUpdateTenant}>
                            <label className="form-field">
                              {t("admin.tenants.table.workspace")}
                              <input className="input-control" value={editingTenant.name} onChange={(e) => setEditingTenant({...editingTenant, name: e.target.value})} required />
                            </label>
                            <label className="form-field">
                            {t("settings.roles.status")}
                              <select className="input-control" value={editingTenant.status} onChange={(e) => setEditingTenant({...editingTenant, status: e.target.value})}>
                                <option value="active">Active</option>
                                <option value="suspended">Suspended</option>
                                <option value="cancelled">Cancelled</option>
                              </select>
                            </label>

                            {/* Statistik & Daftar User */}
                            <div className="surface card shadow-soft spacing-mt-md" style={{ background: "rgba(0,0,0,0.02)", border: "1px solid rgba(0,0,0,0.05)", padding: "1rem" }}>
                              <h4 style={{ fontSize: "0.8rem", textTransform: "uppercase", letterSpacing: "0.05em", marginBottom: "0.75rem", display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                                <span style={{ fontWeight: 700 }}>Daftar User ({editingTenant.user_count})</span>
                                <span style={{ fontSize: "0.7rem", opacity: 0.6, background: "rgba(0,0,0,0.05)", padding: "2px 8px", borderRadius: "10px" }}>Kuota: {editingTenant.quota_users} User</span>
                              </h4>
                              
                              {loadingUsers ? (
                                <div className="text-center-muted py-4">Memuat daftar user...</div>
                              ) : tenantUsers.length === 0 ? (
                                <div className="text-center-muted py-4">Tidak ada user ditemukan.</div>
                              ) : (
                                <div className="table-responsive" style={{ maxHeight: "350px", overflowY: "auto", overflowX: "hidden", borderRadius: "8px", background: "#fff", border: "1px solid rgba(0,0,0,0.05)" }}>
                                  <table className="data-table text-sm">
                                    <thead style={{ position: "sticky", top: 0, zIndex: 1, background: "#f9fafb" }}>
                                      <tr>
                                        <th style={{ padding: "10px 15px" }}>Nama / Email</th>
                                        <th style={{ padding: "10px 15px" }}>Role</th>
                                        <th style={{ padding: "10px 15px" }}>Status</th>
                                        <th style={{ padding: "10px 15px" }} className="text-right">Aksi</th>
                                      </tr>
                                    </thead>
                                    <tbody>
                                      {tenantUsers.map(u => (
                                        <tr key={u.id}>
                                          <td style={{ padding: "10px 15px" }}>
                                            <div style={{ fontWeight: 600 }}>{u.full_name}</div>
                                            <div style={{ fontSize: "0.75rem", opacity: 0.6 }}>{u.email}</div>
                                          </td>
                                          <td style={{ padding: "10px 15px" }}>
                                            <span className="badge-status running-flat" style={{ fontSize: "0.7rem", padding: "2px 8px" }}>
                                              {u.role.toUpperCase()}
                                            </span>
                                          </td>
                                          <td style={{ padding: "10px 15px" }}>
                                            <span className={`badge-dot ${u.status === 'active' ? 'bg-success' : 'bg-danger'}`} style={{ marginRight: "6px" }}></span>
                                            <span style={{ fontSize: "0.8rem", fontWeight: 500 }}>
                                              {u.status === 'active' ? 'Aktif' : 'Non-aktif'}
                                            </span>
                                          </td>
                                          <td style={{ padding: "10px 15px" }} className="text-right">
                                            <div className="table-actions">
                                              <button 
                                                type="button"
                                                className="btn btn-ghost-inline btn-sm" 
                                                onClick={(e) => { 
                                                  e.preventDefault();
                                                  e.stopPropagation();
                                                  setUserToReset({ user: u, action: 'password' }); 
                                                  setResetValue(""); 
                                                }}
                                                title="Reset Password"
                                              >
                                                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 15V17M12 7V13M12 21C16.9706 21 21 16.9706 21 12C21 7.02944 16.9706 3 12 3C7.02944 3 3 7.02944 3 12C3 16.9706 7.02944 21 12 21Z" strokeLinecap="round" strokeLinejoin="round"/></svg>
                                              </button>
                                              <button 
                                                type="button"
                                                className="btn btn-ghost-inline btn-sm" 
                                                onClick={(e) => { 
                                                  e.preventDefault();
                                                  e.stopPropagation();
                                                  setUserToReset({ user: u, action: 'email' }); 
                                                  setResetValue(u.email); 
                                                }}
                                                title="Ubah Email"
                                              >
                                                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2"><path d="M4 4H20C21.1 4 22 4.9 22 6V18C22 19.1 21.1 20 20 20H4C2.9 20 2 19.1 2 18V6C2 4.9 2.9 4 4 4Z" /><polyline points="22,6 12,13 2,6" /></svg>
                                              </button>
                                            </div>
                                          </td>
                                        </tr>
                                      ))}
                                    </tbody>
                                  </table>
                                </div>
                              )}
                            </div>
                            <div className="form-actions spacing-mt-lg">
                              <button className="btn btn-primary" type="submit" disabled={loading}>{t("common.saveChanges")}</button>
                              <button className="btn btn-secondary" type="button" onClick={() => setEditingTenant(null)}>{t("common.cancel")}</button>
                            </div>
                          </form>
                        ) : tenantModalTab === 'backups' ? (
                          <div className="page-section">
                            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1rem" }}>
                              <div>
                                <h4 style={{ margin: 0 }}>Daftar Backup</h4>
                                <p className="text-sm opacity-60">Pencadangan data spesifik untuk workspace ini saja.</p>
                              </div>
                              <button className="btn btn-primary btn-sm" onClick={handleCreateTenantBackup} disabled={loading}>
                                Buat Backup Baru
                              </button>
                            </div>

                            <div className="surface shadow-soft" style={{ background: "#fff", borderRadius: "12px" }}>
                              {loadingBackups ? (
                                <div className="text-center-muted py-4">Memuat daftar backup...</div>
                              ) : tenantBackups.length === 0 ? (
                                <div className="text-center-muted py-4">Belum ada backup untuk workspace ini.</div>
                              ) : (
                                <div className="table-responsive">
                                  <table className="data-table text-sm">
                                    <thead>
                                      <tr>
                                        <th>Nama File</th>
                                        <th>Tanggal</th>
                                        <th>Ukuran</th>
                                        <th className="text-right">Aksi</th>
                                      </tr>
                                    </thead>
                                    <tbody>
                                      {tenantBackups.map(b => (
                                        <tr key={b.name}>
                                          <td><code style={{ fontSize: "0.75rem" }}>{b.name}</code></td>
                                          <td>{new Date(b.created_at).toLocaleString()}</td>
                                          <td>{(b.size / 1024 / 1024).toFixed(2)} MB</td>
                                          <td className="text-right">
                                            <div style={{ display: "flex", gap: "8px", justifyContent: "flex-end" }}>
                                              <button 
                                                className="btn btn-ghost-inline btn-sm" 
                                                onClick={() => handleDownloadTenantBackup(b.name)}
                                                title="Download"
                                              >
                                                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2">
                                                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                                                  <polyline points="7 10 12 15 17 10" />
                                                  <line x1="12" y1="15" x2="12" y2="3" />
                                                </svg>
                                              </button>
                                              <button 
                                                className="btn btn-danger-outline btn-sm" 
                                                onClick={() => handleRestoreTenantBackup(b.name)}
                                                disabled={isRestoring}
                                              >
                                                {isRestoring ? "Restoring..." : "Restore"}
                                              </button>
                                            </div>
                                          </td>
                                        </tr>
                                      ))}
                                    </tbody>
                                  </table>
                                </div>
                              )}
                            </div>
                          </div>
                        ) : tenantModalTab === 'modules' ? (
                          <div className="page-section">
                            <h4 style={{ marginBottom: "1rem" }}>Modul Berlangganan</h4>
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                              {selectedModules.map((m) => (
                                <div key={m.module_code} className="surface card shadow-soft flex justify-between items-center" style={{ padding: "1rem" }}>
                                  <div>
                                    <div style={{ fontWeight: 700 }}>{m.module_code.toUpperCase()}</div>
                                    <div style={{ fontSize: "0.75rem", opacity: 0.6 }}>Aktifkan modul ini untuk tenant.</div>
                                  </div>
                                  <label className="toggle-switch">
                                    <input 
                                      type="checkbox" 
                                      checked={m.is_enabled} 
                                      onChange={() => handleToggleModule(m.module_code, !m.is_enabled)}
                                    />
                                    <span className="slider"></span>
                                  </label>
                                </div>
                              ))}
                            </div>
                          </div>
                        ) : null}
                      </div>

                      {tenantModalTab !== 'info' && (
                        <div className="form-actions spacing-mt-lg" style={{ borderTop: "1px solid var(--border)", paddingTop: "1rem" }}>
                          <button className="btn btn-secondary" type="button" onClick={() => setEditingTenant(null)}>{t("common.close")}</button>
                        </div>
                      )}
                    </div>
                  </div>
                )}
            </>
          )}

          {activeTab === "stats" && stats && (
            <div className="dashboard-view">
              <div className="surface card shadow-soft filter-row-flex">
                <label className="form-field-no-margin">
                  Dari Tanggal
                  <input type="date" className="input-control" value={dateFrom} onChange={(e) => setDateFrom(e.target.value)} />
                </label>
                <label className="form-field-no-margin">
                  Hingga Tanggal
                  <input type="date" className="input-control" value={dateTo} onChange={(e) => setDateTo(e.target.value)} />
                </label>
                <button className="btn btn-secondary" onClick={() => { setDateFrom(""); setDateTo(""); }}>Reset Filter</button>
              </div>

              <div className="card-grid two-col">
                <div className="surface card shadow-soft">
                  <h3 className="form-title">Pertumbuhan Workspace (Historical)</h3>
                  <div ref={statsTenantsRef} className="chart-wrapper-md" />
                </div>
                <div className="surface card shadow-soft">
                  <h3 className="form-title">Pendaftaran User (Historical)</h3>
                  <div ref={statsUsersRef} className="chart-wrapper-md" />
                </div>
              </div>
            </div>
          )}

          {activeTab === "server" && server && (
            <div className="card-grid">
              <div className="surface card shadow-soft">
                <div className="card-header-actions">
                  <h3 className="form-title">Kesehatan Infrastruktur</h3>
                  <span className="badge-status running text-xs-caps">Monitor Real-time {t("settings.users.statusActive")}</span>
                </div>
                <div className="info-grid-two-col">
                   <div className="info-section">
                      <h4 className="info-section-title">Sistem Operasi</h4>
                      <div className="info-list">
                        <div className="info-list-item"><strong>OS:</strong> <code>{server.os}</code></div>
                        <div className="info-list-item"><strong>Uptime:</strong> {server.uptime}</div>
                        <div className="info-list-item"><strong>IP Server:</strong> <code>{server.ip_address}</code></div>
                        <div className="info-list-item"><strong>Port HTTP:</strong> <code>{server.port}</code></div>
                      </div>
                   </div>
                   <div className="info-section">
                      <h4 className="info-section-title">Basis Data & Cache</h4>
                      <div className="info-list">
                        <div className="info-list-item"><strong>PostgreSQL:</strong> <span className={`badge-status ${server.db_status.includes("Healthy") ? "running" : "error"}`}>{server.db_status}</span></div>
                        <div className="info-list-item"><strong>Redis:</strong> <span className="badge-status running">{server.redis_status}</span></div>
                      </div>
                   </div>
                </div>
                
                <h4 className="info-section-title spacing-mt-lg">{t("admin.server.services")}</h4>
                <div className="data-table-wrap table-mobile-stack">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>{t("admin.server.service_name")}</th>
                        <th>{t("admin.server.status")}</th>
                        <th className="text-right">{t("admin.server.port")}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(server?.services || []).map((s, i) => (
                        <tr key={i}>
                          <td className="font-bold">{s.name}</td>
                          <td><span className="badge-status running">{s.status}</span></td>
                          <td className="text-right"><code>{s.port || "-"}</code></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          )}

          {activeTab === "logs" && (
            <div className="surface card shadow-soft">
              <h3 className="form-title">{t("admin.logs.title")}</h3>
              <div className="data-table-wrap table-mobile-stack">
                <table className="data-table text-sm">
                  <thead>
                    <tr>
                      <th>{t("common.timestamp")}</th>
                      <th>Workspace</th>
                      <th>Actor</th>
                      <th>{t("settings.audit.action")}</th>
                      <th>{t("settings.audit.resource")}</th>
                      <th>Details</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(logs || []).length === 0 && !loading && <tr><td colSpan={6} className="text-center-muted">{t("common.noData")}</td></tr>}
                    {(logs || []).map(l => (
                      <tr key={l.id}>
                        <td className="no-wrap">{new Date(l.created_at).toLocaleString()}</td>
                        <td>
                          <strong>{l.tenant_name || "System"}</strong><br/>
                          <code className="text-xs">{l.tenant_id ? l.tenant_id.slice(0,8) : "-"}</code>
                        </td>
                        <td>
                          <strong>{l.actor_user_name || "Unknown"}</strong><br/>
                          <code className="text-xs">{l.ip_address}</code>
                        </td>
                        <td><span className="badge-status running-flat">{l.action}</span></td>
                        <td className="opacity-90">{l.resource}</td>
                        <td className="opacity-80">
                          {l.details || (l.resource_id ? `ID: ${l.resource_id.slice(0,8)}...` : "-")}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {activeTab === "notifications" && (
            <div className="card-grid two-col tight">
               {/* Global Provider Selection */}
               <div className="surface card shadow-soft" style={{ gridColumn: "1 / -1" }}>
                 <h3 className="form-title">Provider WhatsApp Aktif</h3>
                 <div className="form-grid">
                    <p className="form-section-desc">Pilih provider mana yang akan digunakan secara global untuk pengiriman OTP (Pendaftaran/Lupa Password).</p>
                    <div style={{ display: "flex", gap: "10px", alignItems: "center" }}>
                      <select 
                        className="input-control" 
                        value={waActiveProvider} 
                        onChange={e => setWaActiveProvider(e.target.value)} 
                        style={{ maxWidth: "300px" }}
                      >
                        <option value="wa">WhatsApp Official (Meta)</option>
                        <option value="wa_fonnte">WhatsApp Fonnte Provider</option>
                        <option value="wa_waha">WhatsApp WAHA Provider</option>
                        <option value="wa_gowa">WhatsApp GOWA Provider</option>
                      </select>
                      <button className="btn btn-primary" onClick={() => handleSaveNotificationConfig("wa_active_provider")} disabled={savingConfig === "wa_active_provider"}>
                        {savingConfig === "wa_active_provider" ? "Menyimpan..." : "Simpan Pilihan"}
                      </button>
                    </div>
                 </div>
               </div>

               <div className="surface card shadow-soft">
                 <h3 className="form-title">SMTP Email Provider (Secured)</h3>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveNotificationConfig("smtp"); }}>
                    <label className="form-field">Host <input className="input-control" placeholder="smtp.gmail.com" value={smtpConfig.host} onChange={e => setSmtpConfig({...smtpConfig, host: e.target.value})} /></label>
                    <label className="form-field">Port <input className="input-control" placeholder="587" value={smtpConfig.port} onChange={e => setSmtpConfig({...smtpConfig, port: e.target.value})} /></label>
                    <label className="form-field">Username <input className="input-control" placeholder="user@gmail.com" value={smtpConfig.username} onChange={e => setSmtpConfig({...smtpConfig, username: e.target.value})} /></label>
                    <label className="form-field">Password <PasswordInput placeholder="••••••••" value={smtpConfig.password} onChange={e => setSmtpConfig({...smtpConfig, password: e.target.value})} /></label>
                    <label className="form-field">
                      Security
                      <select 
                        className="input-control" 
                        value={smtpConfig.security} 
                        onChange={e => setSmtpConfig({...smtpConfig, security: e.target.value})}
                      >
                        <option value="none">None (Standard)</option>
                        <option value="ssl">SSL / TLS (Port 465)</option>
                        <option value="starttls">STARTTLS (Port 587)</option>
                      </select>
                    </label>
                    <div className="form-actions-inline" style={{ display: "flex", gap: "10px" }}>
                      <button className="btn btn-primary" type="submit" disabled={savingConfig === "smtp"}>
                        {savingConfig === "smtp" ? t("common.loading") : t("admin.notifications.save")}
                      </button>
                      <button className="btn btn-secondary-outline" type="button" onClick={() => handleTestConnection("smtp")} disabled={testingConfig === "smtp"}>
                        {testingConfig === "smtp" ? "Testing..." : "Test Connection"}
                      </button>
                    </div>
                 </form>
               </div>
               <div className="surface card shadow-soft">
                 <h3 className="form-title">Telegram Bot Provider</h3>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveNotificationConfig("telegram"); }}>
                    <label className="form-field">Bot Token <PasswordInput placeholder="000000:ABC-DEF..." value={telegramConfig.botToken} onChange={e => setTelegramConfig({...telegramConfig, botToken: e.target.value})} /></label>
                    <label className="form-field">Bot Name <input className="input-control" placeholder="@PekanBot" value={telegramConfig.botName} onChange={e => setTelegramConfig({...telegramConfig, botName: e.target.value})} /></label>
                    <div className="form-actions-inline" style={{ display: "flex", gap: "10px" }}>
                      <button className="btn btn-primary" type="submit" disabled={savingConfig === "telegram"}>
                        {savingConfig === "telegram" ? t("common.loading") : t("admin.notifications.save")}
                      </button>
                      <button className="btn btn-secondary-outline" type="button" onClick={() => handleTestConnection("telegram")} disabled={testingConfig === "telegram"}>
                        {testingConfig === "telegram" ? "Testing..." : "Test Connection"}
                      </button>
                    </div>
                 </form>
               </div>
               <div className="surface card shadow-soft">
                 <h3 className="form-title">WhatsApp Official (Meta)</h3>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveNotificationConfig("wa"); }}>
                    <label className="form-field">API Token <PasswordInput placeholder="EAAB..." value={waConfig.apiToken} onChange={e => setWaConfig({...waConfig, apiToken: e.target.value})} /></label>
                    <label className="form-field">Phone ID <input className="input-control" placeholder="102938..." value={waConfig.phoneId} onChange={e => setWaConfig({...waConfig, phoneId: e.target.value})} /></label>
                    <div className="form-actions-inline" style={{ display: "flex", gap: "10px" }}>
                      <button className="btn btn-primary" type="submit" disabled={savingConfig === "wa"}>
                        {savingConfig === "wa" ? t("common.loading") : t("admin.notifications.save")}
                      </button>
                      <button className="btn btn-secondary-outline" type="button" onClick={() => handleTestConnection("wa")} disabled={testingConfig === "wa"}>
                        {testingConfig === "wa" ? "Testing..." : "Test Connection"}
                      </button>
                    </div>
                 </form>
               </div>
               <div className="surface card shadow-soft">
                 <h3 className="form-title">WhatsApp Fonnte Provider</h3>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveNotificationConfig("wa_fonnte"); }}>
                    <label className="form-field">API Key <PasswordInput placeholder="FONNTE_KEY..." value={waFonnteConfig.apiKey} onChange={e => setWaFonnteConfig({apiKey: e.target.value})} /></label>
                    <div className="form-actions-inline" style={{ display: "flex", gap: "10px" }}>
                      <button className="btn btn-primary" type="submit" disabled={savingConfig === "wa_fonnte"}>
                        {savingConfig === "wa_fonnte" ? t("common.loading") : t("admin.notifications.save")}
                      </button>
                      <button className="btn btn-secondary-outline" type="button" onClick={() => handleTestConnection("wa_fonnte")} disabled={testingConfig === "wa_fonnte"}>
                        {testingConfig === "wa_fonnte" ? "Testing..." : "Test Connection"}
                      </button>
                    </div>
                 </form>
               </div>
               <div className="surface card shadow-soft">
                 <h3 className="form-title">WhatsApp WAHA Provider</h3>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveNotificationConfig("wa_waha"); }}>
                    <div className="card-grid two-col" style={{ gap: "10px" }}>
                      <label className="form-field">API URL <input className="input-control" placeholder="https://waha.domain.com" value={waWahaConfig.apiUrl} onChange={e => setWaWahaConfig({...waWahaConfig, apiUrl: e.target.value})} /></label>
                      <label className="form-field">Session Name <input className="input-control" placeholder="default" value={waWahaConfig.session} onChange={e => setWaWahaConfig({...waWahaConfig, session: e.target.value})} /></label>
                    </div>
                    <label className="form-field">API Key <PasswordInput placeholder="WAHA_SECRET..." value={waWahaConfig.apiKey} onChange={e => setWaWahaConfig({...waWahaConfig, apiKey: e.target.value})} /></label>
                    <div className="form-actions-inline" style={{ display: "flex", gap: "10px" }}>
                      <button className="btn btn-primary" type="submit" disabled={savingConfig === "wa_waha"}>
                        {savingConfig === "wa_waha" ? t("common.loading") : t("admin.notifications.save")}
                      </button>
                      <button className="btn btn-secondary-outline" type="button" onClick={() => handleTestConnection("wa_waha")} disabled={testingConfig === "wa_waha"}>
                        {testingConfig === "wa_waha" ? "Testing..." : "Test Connection"}
                      </button>
                    </div>
                 </form>
               </div>
               <div className="surface card shadow-soft">
                 <h3 className="form-title">WhatsApp GOWA Provider</h3>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveNotificationConfig("wa_gowa"); }}>
                    <label className="form-field">API Key <PasswordInput placeholder="GOWA_KEY..." value={waGowaConfig.apiKey} onChange={e => setWaGowaConfig({apiKey: e.target.value})} /></label>
                    <div className="form-actions-inline" style={{ display: "flex", gap: "10px" }}>
                      <button className="btn btn-primary" type="submit" disabled={savingConfig === "wa_gowa"}>
                        {savingConfig === "wa_gowa" ? t("common.loading") : t("admin.notifications.save")}
                      </button>
                      <button className="btn btn-secondary-outline" type="button" onClick={() => handleTestConnection("wa_gowa")} disabled={testingConfig === "wa_gowa"}>
                        {testingConfig === "wa_gowa" ? "Testing..." : "Test Connection"}
                      </button>
                    </div>
                 </form>
               </div>
            </div>
          )}

          {activeTab === "ai" && (
            <div className="card-grid tight">
               {/* Active AI Provider Selection for Scan Receipt */}
               <div className="surface card shadow-soft" style={{ gridColumn: "1 / -1" }}>
                 <h3 className="form-title">Provider AI Aktif (Scan Receipt)</h3>
                 <div className="form-grid">
                    <p className="form-section-desc">Pilih provider mana yang akan digunakan secara global untuk fitur Scan Receipt.</p>
                    <div style={{ display: "flex", gap: "10px", alignItems: "center" }}>
                      <select className="input-control" value={activeAI} onChange={e => setActiveAI(e.target.value)} style={{ maxWidth: "300px" }}>
                        <option value="gemini">Google Gemini</option>
                        <option value="openai">OpenAI (ChatGPT)</option>
                        <option value="claude">Anthropic Claude</option>
                        <option value="sumopod">Sumopod (Custom)</option>
                      </select>
                      <button className="btn btn-primary" onClick={() => handleSaveNotificationConfig("active_ai")} disabled={savingConfig === "active_ai"}>
                        {savingConfig === "active_ai" ? "Menyimpan..." : "Simpan Pilihan"}
                      </button>
                    </div>
                 </div>
               </div>

               {/* Active AI Provider Selection for Chat Bot AI */}
               <div className="surface card shadow-soft" style={{ gridColumn: "1 / -1" }}>
                 <h3 className="form-title">Provider AI Aktif untuk Chat Bot AI (WhatsApp)</h3>
                 <div className="form-grid">
                    <p className="form-section-desc">Pilih provider mana yang akan digunakan secara global untuk fitur Chat Bot AI WhatsApp.</p>
                    <div style={{ display: "flex", gap: "10px", alignItems: "center" }}>
                      <select className="input-control" value={activeWaBotAI} onChange={e => setActiveWaBotAI(e.target.value)} style={{ maxWidth: "300px" }}>
                        <option value="gemini">Google Gemini</option>
                        <option value="openai">OpenAI (ChatGPT)</option>
                        <option value="claude">Anthropic Claude</option>
                        <option value="sumopod">Sumopod (Custom)</option>
                      </select>
                      <button className="btn btn-primary" onClick={() => handleSaveNotificationConfig("active_wa_bot_ai")} disabled={savingConfig === "active_wa_bot_ai"}>
                        {savingConfig === "active_wa_bot_ai" ? "Menyimpan..." : "Simpan Pilihan"}
                      </button>
                    </div>
                 </div>
               </div>
 
               {/* Nomor WhatsApp Bot System */}
               <div className="surface card shadow-soft" style={{ gridColumn: "1 / -1" }}>
                 <h3 className="form-title">Nomor WhatsApp Bot System</h3>
                 <div className="form-grid">
                    <p className="form-section-desc">Masukkan nomor WhatsApp Bot sistem Anda (misal: +628123456789). Nomor ini akan ditampilkan kepada pengguna untuk membantu mereka mengetahui kemana mereka harus mengirim pesan/perintah login.</p>
                    <div style={{ display: "flex", gap: "10px", alignItems: "center" }}>
                      <input 
                        type="text"
                        className="input-control" 
                        placeholder="Contoh: +628123456789" 
                        value={waBotPhoneNumber} 
                        onChange={e => setWaBotPhoneNumber(e.target.value)} 
                        style={{ maxWidth: "300px" }} 
                      />
                      <button className="btn btn-primary" onClick={() => handleSaveNotificationConfig("wa_bot_phone_number")} disabled={savingConfig === "wa_bot_phone_number"}>
                        {savingConfig === "wa_bot_phone_number" ? "Menyimpan..." : "Simpan Nomor Bot"}
                      </button>
                    </div>
                 </div>
               </div>

                {/* Chat Bot System Prompt Tuning */}
                <div className="surface card shadow-soft" style={{ gridColumn: "1 / -1" }}>
                  <h3 className="form-title">Tuning Setup / System Prompt Chat Bot AI (WhatsApp)</h3>
                  <div className="form-grid">
                     <p className="form-section-desc">Sesuaikan instruksi khusus, kepribadian, format balasan, dan perilaku asisten keuangan WhatsApp Anda di sini secara fleksibel. (Data transaksi riil rincian bulanan, anggaran, dan daftar riwayat transaksi terbaru akan otomatis ditambahkan di bagian bawah oleh sistem).</p>
                     <div style={{ display: "flex", flexDirection: "column", gap: "12px", width: "100%" }}>
                       <textarea 
                         className="input-control" 
                         value={waBotSystemPrompt} 
                         onChange={e => setWaBotSystemPrompt(e.target.value)} 
                         style={{ width: "100%", minHeight: "220px", fontFamily: "monospace", fontSize: "14px", lineHeight: "1.5" }}
                         placeholder="Masukkan instruksi khusus untuk asisten AI..."
                       />
                       <div style={{ display: "flex", gap: "10px" }}>
                         <button className="btn btn-primary" onClick={() => handleSaveNotificationConfig("wa_bot_system_prompt")} disabled={savingConfig === "wa_bot_system_prompt"}>
                           {savingConfig === "wa_bot_system_prompt" ? "Menyimpan..." : "Simpan System Prompt"}
                         </button>
                         <button className="btn btn-secondary" onClick={() => setWaBotSystemPrompt(DEFAULT_WA_BOT_SYSTEM_PROMPT)}>
                           Reset ke Default
                         </button>
                       </div>
                     </div>
                  </div>
                </div>

                {/* Gemini Config */}
               <div className="surface card shadow-soft">
                 <div className="form-header-with-badge">
                   <h3 className="form-title">Google Gemini</h3>
                   <div style={{ display: "flex", gap: "5px" }}>
                     {activeAI === "gemini" && <span className="badge-status running">SCAN RECEIPT</span>}
                     {activeWaBotAI === "gemini" && <span className="badge-status stopped">CHAT BOT</span>}
                   </div>
                 </div>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveNotificationConfig("gemini"); }}>
                    <label className="form-field">API Key <PasswordInput placeholder="AIza..." value={geminiConfig.apiKey} onChange={e => setGeminiConfig({...geminiConfig, apiKey: e.target.value})} /></label>
                    <label className="form-field">
                      Model Default
                      <div style={{ display: "flex", gap: "10px" }}>
                        <select className="input-control" value={geminiConfig.model} onChange={e => setGeminiConfig({...geminiConfig, model: e.target.value})}>
                          {(aiModels || []).map(m => <option key={m.id} value={m.id}>{m.label}</option>)}
                        </select>
                      </div>
                    </label>
                    <div className="form-actions-inline" style={{ display: "flex", gap: "10px" }}>
                      <button className="btn btn-primary" type="submit" disabled={savingConfig === "gemini"}>Simpan</button>
                      <button className="btn btn-secondary-outline" type="button" onClick={() => handleTestConnection("gemini")} disabled={testingConfig === "gemini"}>Test & Fetch</button>
                    </div>
                 </form>
               </div>

               {/* OpenAI Config */}
               <div className="surface card shadow-soft">
                 <div className="form-header-with-badge">
                   <h3 className="form-title">OpenAI (ChatGPT)</h3>
                   <div style={{ display: "flex", gap: "5px" }}>
                     {activeAI === "openai" && <span className="badge-status running">SCAN RECEIPT</span>}
                     {activeWaBotAI === "openai" && <span className="badge-status stopped">CHAT BOT</span>}
                   </div>
                 </div>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveNotificationConfig("openai"); }}>
                    <label className="form-field">API Key <PasswordInput placeholder="sk-..." value={openaiConfig.apiKey} onChange={e => setOpenaiConfig({...openaiConfig, apiKey: e.target.value})} /></label>
                    <label className="form-field">
                      Model Default
                      <select className="input-control" value={openaiConfig.model} onChange={e => setOpenaiConfig({...openaiConfig, model: e.target.value})}>
                        {(openaiModels || []).map(m => <option key={m.id} value={m.id}>{m.label}</option>)}
                      </select>
                    </label>
                    <div className="form-actions-inline" style={{ display: "flex", gap: "10px" }}>
                      <button className="btn btn-primary" type="submit" disabled={savingConfig === "openai"}>Simpan</button>
                      <button className="btn btn-secondary-outline" type="button" onClick={() => handleTestConnection("openai")} disabled={testingConfig === "openai"}>Test & Fetch</button>
                    </div>
                 </form>
               </div>

               {/* Claude Config */}
               <div className="surface card shadow-soft">
                 <div className="form-header-with-badge">
                   <h3 className="form-title">Anthropic Claude</h3>
                   <div style={{ display: "flex", gap: "5px" }}>
                     {activeAI === "claude" && <span className="badge-status running">SCAN RECEIPT</span>}
                     {activeWaBotAI === "claude" && <span className="badge-status stopped">CHAT BOT</span>}
                   </div>
                 </div>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveNotificationConfig("claude"); }}>
                    <label className="form-field">API Key <PasswordInput placeholder="sk-ant-..." value={claudeConfig.apiKey} onChange={e => setClaudeConfig({...claudeConfig, apiKey: e.target.value})} /></label>
                    <label className="form-field">
                      Model Default
                      <select className="input-control" value={claudeConfig.model} onChange={e => setClaudeConfig({...claudeConfig, model: e.target.value})}>
                        {(claudeModels || []).map(m => <option key={m.id} value={m.id}>{m.label}</option>)}
                      </select>
                    </label>
                    <div className="form-actions-inline" style={{ display: "flex", gap: "10px" }}>
                      <button className="btn btn-primary" type="submit" disabled={savingConfig === "claude"}>Simpan</button>
                      <button className="btn btn-secondary-outline" type="button" onClick={() => handleTestConnection("claude")} disabled={testingConfig === "claude"}>Test Connection</button>
                    </div>
                 </form>
               </div>

               {/* Sumopod Config */}
               <div className="surface card shadow-soft">
                 <div className="form-header-with-badge">
                   <h3 className="form-title">Sumopod (Custom)</h3>
                   <div style={{ display: "flex", gap: "5px" }}>
                     {activeAI === "sumopod" && <span className="badge-status running">SCAN RECEIPT</span>}
                     {activeWaBotAI === "sumopod" && <span className="badge-status stopped">CHAT BOT</span>}
                   </div>
                 </div>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveNotificationConfig("sumopod"); }}>
                    <label className="form-field">API Key / Token <PasswordInput placeholder="SUMO-..." value={sumopodConfig.apiKey} onChange={e => setSumopodConfig({...sumopodConfig, apiKey: e.target.value})} /></label>
                    <label className="form-field">
                      Model ID 
                      {sumopodModels.length > 0 ? (
                        <select className="input-control" value={sumopodConfig.model} onChange={e => setSumopodConfig({...sumopodConfig, model: e.target.value})}>
                          {sumopodModels.map(m => <option key={m.id} value={m.id}>{m.label || m.id}</option>)}
                        </select>
                      ) : (
                        <input className="input-control" placeholder="Klik Test & Fetch Models" value={sumopodConfig.model} onChange={e => setSumopodConfig({...sumopodConfig, model: e.target.value})} />
                      )}
                    </label>
                    <div className="form-actions-inline" style={{ display: "flex", gap: "10px" }}>
                      <button className="btn btn-primary" type="submit" disabled={savingConfig === "sumopod"}>Simpan</button>
                      <button className="btn btn-secondary-outline" type="button" onClick={() => handleTestConnection("sumopod")} disabled={testingConfig === "sumopod"}>Test & Fetch Models</button>
                    </div>
                 </form>
               </div>
            </div>
          )}

          {activeTab === "database" && (
            <div className="card-grid tight">
               <div className="surface card shadow-soft" style={{ gridColumn: "1 / -1" }}>
                 <h3 className="form-title">PostgreSQL Database Configuration</h3>
                 <p className="form-section-desc" style={{ marginBottom: "1rem" }}>
                   Pengaturan koneksi ke database PostgreSQL utama. Mengonfigurasi ini tidak akan mengubah koneksi instance yang sedang berjalan saat ini, 
                   melainkan menyimpannya ke <strong>Global Settings</strong> untuk keperluan integrasi HA (High Availability) atau Multi-database node di masa mendatang.
                 </p>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveDbConfig(); }}>
                    <div className="form-row" style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "15px" }}>
                      <label className="form-field">
                        Host
                        <input className="input-control" placeholder="127.0.0.1" value={dbConfig.host} onChange={e => setDbConfig({...dbConfig, host: e.target.value})} required />
                      </label>
                      <label className="form-field">
                        Port
                        <input className="input-control" placeholder="5432" value={dbConfig.port} onChange={e => setDbConfig({...dbConfig, port: e.target.value})} required />
                      </label>
                    </div>
                    <label className="form-field">
                      Database Name
                      <input className="input-control" placeholder="pekan" value={dbConfig.dbname} onChange={e => setDbConfig({...dbConfig, dbname: e.target.value})} required />
                    </label>
                    <div className="form-row" style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "15px" }}>
                      <label className="form-field">
                        User
                        <input className="input-control" placeholder="postgres" value={dbConfig.user} onChange={e => setDbConfig({...dbConfig, user: e.target.value})} required />
                      </label>
                      <label className="form-field">
                        Password
                        <PasswordInput placeholder="••••••••" value={dbConfig.password} onChange={e => setDbConfig({...dbConfig, password: e.target.value})} />
                      </label>
                    </div>
                    <label className="form-field">
                      SSL Mode
                      <select className="input-control" value={dbConfig.sslmode} onChange={e => setDbConfig({...dbConfig, sslmode: e.target.value})}>
                        <option value="disable">Disable</option>
                        <option value="require">Require</option>
                        <option value="verify-ca">Verify CA</option>
                        <option value="verify-full">Verify Full</option>
                      </select>
                    </label>
                    <div className="form-actions-inline" style={{ display: "flex", gap: "10px", marginTop: "10px" }}>
                      <button className="btn btn-primary" type="submit" disabled={savingConfig === "database"}>
                        {savingConfig === "database" ? "Menyimpan..." : "Save Configuration"}
                      </button>
                      <button className="btn btn-secondary-outline" type="button" onClick={handleTestDbConfig} disabled={testingConfig === "database"}>
                        {testingConfig === "database" ? "Testing..." : "Test Connection"}
                      </button>
                    </div>
                 </form>
               </div>
            </div>
          )}

          {activeTab === "backups" && (
            <div className="card-grid tight">
               <div className="surface card shadow-soft" style={{ gridColumn: "1 / -1" }}>
                 <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1.5rem" }}>
                   <h3 className="form-title" style={{ margin: 0 }}>Daftar File Backup</h3>
                   <div style={{ display: "flex", gap: "10px", alignItems: "center" }}>
                     <select className="input-control" value={backupType} onChange={e => setBackupType(e.target.value)}>
                       <option value="full">Full Backup</option>
                       <option value="schema">Schema Only</option>
                       <option value="data">Data Only</option>
                     </select>
                     <button className="btn btn-primary" onClick={handleCreateBackup} disabled={isBackingUp || loading} style={{ whiteSpace: "nowrap" }}>
                       {isBackingUp ? "Membuat Backup..." : "+ Buat Backup"}
                     </button>
                   </div>
                 </div>

                 {loading ? (
                   <div className="empty-state">Memuat data...</div>
                 ) : (backups || []).length === 0 ? (
                   <div className="empty-state">Belum ada file backup yang dibuat.</div>
                 ) : (
                   <div className="table-responsive">
                     <table className="data-table">
                       <thead>
                         <tr>
                           <th>Nama File</th>
                           <th>Ukuran</th>
                           <th>Tanggal Dibuat</th>
                           <th>Aksi</th>
                         </tr>
                       </thead>
                       <tbody>
                         {(backups || []).map(b => (
                           <tr key={b.name}>
                             <td><strong>{b.name}</strong></td>
                             <td>{(b.size / 1024 / 1024).toFixed(2)} MB</td>
                             <td>{new Date(b.created_at).toLocaleString(locale)}</td>
                             <td>
                               <div className="table-actions">
                                 <button className="btn btn-ghost-inline btn-sm" onClick={() => handleDownloadBackup(b.name)} aria-label="Download" title="Download">
                                   <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                                     <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                                     <polyline points="7 10 12 15 17 10" />
                                     <line x1="12" y1="15" x2="12" y2="3" />
                                   </svg>
                                 </button>
                                 <button className="btn btn-ghost-inline btn-sm danger" onClick={() => setFileToRestore(b.name)} aria-label="Restore" title="Restore (WARNING)">
                                   <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                                     <polyline points="1 4 1 10 7 10" />
                                     <path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10" />
                                   </svg>
                                 </button>
                               </div>
                             </td>
                           </tr>
                         ))}
                       </tbody>
                      </table>
                    </div>
                  )}
                </div>
            </div>
          )}

          {activeTab === "optimization" && (
            <div className="card-grid tight">
               <div className="surface card shadow-soft" style={{ gridColumn: "1 / -1" }}>
                  <h3 className="form-title">Optimasi Backend & Konfigurasi Sistem</h3>
                  <p className="form-section-desc">Pengaturan ini disimpan di database dan akan diprioritaskan saat backend dijalankan ulang.</p>
                  
                  <form className="form-grid spacing-mt-lg" onSubmit={(e) => { e.preventDefault(); handleSaveOptimization(); }}>
                    <div className="form-row" style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "20px" }}>
                      <label className="form-field">
                        API Rate Limit (RPM)
                        <input className="input-control" type="number" placeholder="2000" value={optConfig.api_rate_limit} onChange={e => setOptConfig({...optConfig, api_rate_limit: e.target.value})} />
                        <span className="opacity-50 text-xs">Jumlah permintaan maksimal per IP per menit.</span>
                      </label>
                      <label className="form-field">
                        API Request Timeout
                        <input className="input-control" placeholder="30s" value={optConfig.api_timeout} onChange={e => setOptConfig({...optConfig, api_timeout: e.target.value})} />
                        <span className="opacity-50 text-xs">Batas waktu tunggu server sebelum memutus koneksi.</span>
                      </label>
                    </div>
                    
                    <label className="form-field">
                       Max Request Body Size
                       <input className="input-control" placeholder="10mb" value={optConfig.max_upload_size} onChange={e => setOptConfig({...optConfig, max_upload_size: e.target.value})} />
                       <span className="opacity-50 text-xs">Batas ukuran file/payload (misal: 10mb, 50mb).</span>
                    </label>

                    <div className="alert alert-info spacing-mt-md">
                       <strong>Catatan:</strong> Perubahan pada Rate Limit dan Timeout mungkin memerlukan restart container/backend untuk benar-benar diterapkan secara menyeluruh ke seluruh thread.
                    </div>

                    <button className="btn btn-primary spacing-mt-lg" type="submit" disabled={savingConfig === "active_ai"}>
                      {savingConfig === "active_ai" ? "Menyimpan..." : "Simpan Pengaturan Optimasi"}
                    </button>
                  </form>
               </div>
            </div>
          )}

          {activeTab === "dbtool" && (
            <div className="card-grid tight">

               <div className="surface card shadow-soft" style={{ gridColumn: "1 / -1" }}>
                  <div className="card-header-actions">
                    <h3 className="form-title">Visualisasi Pertumbuhan Database</h3>
                    <button className="btn btn-ghost-inline btn-sm" onClick={loadDbGrowth}>Refresh Chart</button>
                  </div>
                  <div ref={dbGrowthRef} style={{ height: "350px", width: "100%" }}></div>
               </div>

               <div className="surface card shadow-soft" style={{ gridColumn: "1 / -1" }}>
                 <div className="card-header-actions">
                   <h3 className="form-title">Statistik Ukuran Tabel</h3>
                   <button className="btn btn-ghost-inline btn-sm" onClick={loadDbStats}>Refresh Stats</button>
                 </div>
                 <div className="table-responsive">
                    <table className="data-table">
                      <thead>
                        <tr>
                          <th>Table Name</th>
                          <th>Rows</th>
                          <th>Data Size</th>
                          <th>Index Size</th>
                          <th>Total Size</th>
                        </tr>
                      </thead>
                      <tbody>
                        {dbStats.map(s => (
                          <tr key={s.name}>
                            <td><strong>{s.name}</strong></td>
                            <td>{s.rows.toLocaleString()}</td>
                            <td>{s.data_size}</td>
                            <td>{s.index_size}</td>
                            <td>{s.total_size}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                 </div>
               </div>
            </div>
          )}

          {activeTab === "whatsapp" && (
            <div className="whatsapp-queue-view">
              {/* KPI Metrics */}
              <div 
                className="card-grid spacing-mb-lg" 
                style={{ 
                  gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))", 
                  gap: "1.25rem",
                  marginBottom: "2rem" 
                }}
              >
                {/* CARD 1: Total Pesan Diproses */}
                <div 
                  className="surface card shadow-soft stat-card"
                  style={{
                    position: "relative",
                    background: "linear-gradient(135deg, var(--surface) 0%, rgba(99, 102, 241, 0.05) 100%)",
                    borderLeft: "4px solid #6366f1",
                    transition: "transform 0.2s, box-shadow 0.2s",
                    display: "flex",
                    flexDirection: "column",
                    justifyContent: "space-between",
                    padding: "1.5rem"
                  }}
                  onMouseEnter={e => {
                    e.currentTarget.style.transform = "translateY(-2px)";
                    e.currentTarget.style.boxShadow = "var(--shadow-lg)";
                  }}
                  onMouseLeave={e => {
                    e.currentTarget.style.transform = "none";
                    e.currentTarget.style.boxShadow = "var(--shadow-soft)";
                  }}
                >
                  <div>
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "0.5rem" }}>
                      <p className="stat-label" style={{ fontWeight: 600, color: "var(--muted)", margin: 0, textTransform: "uppercase", fontSize: "0.75rem", letterSpacing: "0.05em" }}>
                        Total Pesan Diproses
                      </p>
                    </div>
                    <h2 className="stat-value text-indigo" style={{ fontSize: "2rem", fontWeight: 800, margin: "0.25rem 0", color: "#6366f1" }}>
                      {waQueueStats?.total_processed?.toLocaleString() || 0}
                    </h2>
                  </div>
                  <p className="stat-meta" style={{ margin: "0.5rem 0 0", fontSize: "0.85rem", display: "flex", gap: "8px", alignItems: "center" }}>
                    <span className="badge bg-success-soft text-emerald" style={{ padding: "2px 8px", borderRadius: "12px", background: "rgba(16, 185, 129, 0.1)", fontWeight: 600 }}>
                      {(waQueueStats?.total_success || 0).toLocaleString()} Sukses
                    </span>
                    <span className="badge bg-danger-soft text-rose" style={{ padding: "2px 8px", borderRadius: "12px", background: "rgba(239, 68, 68, 0.1)", fontWeight: 600 }}>
                      {(waQueueStats?.total_failed || 0).toLocaleString()} Gagal
                    </span>
                  </p>
                </div>

                {/* CARD 2: Dalam Antrean (Pending) */}
                <div 
                  className="surface card shadow-soft stat-card"
                  style={{
                    position: "relative",
                    background: "linear-gradient(135deg, var(--surface) 0%, rgba(245, 158, 11, 0.05) 100%)",
                    borderLeft: "4px solid #f59e0b",
                    transition: "transform 0.2s, box-shadow 0.2s",
                    display: "flex",
                    flexDirection: "column",
                    justifyContent: "space-between",
                    padding: "1.5rem"
                  }}
                  onMouseEnter={e => {
                    e.currentTarget.style.transform = "translateY(-2px)";
                    e.currentTarget.style.boxShadow = "var(--shadow-lg)";
                  }}
                  onMouseLeave={e => {
                    e.currentTarget.style.transform = "none";
                    e.currentTarget.style.boxShadow = "var(--shadow-soft)";
                  }}
                >
                  <div>
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "0.5rem" }}>
                      <p className="stat-label" style={{ fontWeight: 600, color: "var(--muted)", margin: 0, textTransform: "uppercase", fontSize: "0.75rem", letterSpacing: "0.05em" }}>
                        Dalam Antrean (Pending)
                      </p>
                    </div>
                    <h2 className={`stat-value ${waQueueStats?.total_pending && waQueueStats.total_pending > 0 ? "text-amber pulse-fast" : "text-muted"}`} style={{ fontSize: "2rem", fontWeight: 800, margin: "0.25rem 0", color: waQueueStats?.total_pending && waQueueStats.total_pending > 0 ? "#f59e0b" : "var(--muted)" }}>
                      {waQueueStats?.total_pending || 0}
                    </h2>
                  </div>
                  <p className="stat-meta" style={{ margin: "0.5rem 0 0", fontSize: "0.82rem", color: "var(--muted)" }}>
                    {waQueueStats?.total_pending && waQueueStats.total_pending > 0 ? (
                      <span style={{ display: "flex", alignItems: "center", gap: "6px" }}>
                        <span className="badge-dot bg-warning pulse-fast" style={{ width: "8px", height: "8px", borderRadius: "50%", background: "#f59e0b", display: "inline-block" }}></span>
                        Menunggu pemrosesan worker...
                      </span>
                    ) : "Antrean kosong & sehat"}
                  </p>
                </div>

                {/* CARD 3: Sedang Diproses (Active AI) */}
                <div 
                  className="surface card shadow-soft stat-card"
                  style={{
                    position: "relative",
                    background: "linear-gradient(135deg, var(--surface) 0%, rgba(99, 102, 241, 0.05) 100%)",
                    borderLeft: "4px solid #4f46e5",
                    transition: "transform 0.2s, box-shadow 0.2s",
                    display: "flex",
                    flexDirection: "column",
                    justifyContent: "space-between",
                    padding: "1.5rem"
                  }}
                  onMouseEnter={e => {
                    e.currentTarget.style.transform = "translateY(-2px)";
                    e.currentTarget.style.boxShadow = "var(--shadow-lg)";
                  }}
                  onMouseLeave={e => {
                    e.currentTarget.style.transform = "none";
                    e.currentTarget.style.boxShadow = "var(--shadow-soft)";
                  }}
                >
                  <div>
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "0.5rem" }}>
                      <p className="stat-label" style={{ fontWeight: 600, color: "var(--muted)", margin: 0, textTransform: "uppercase", fontSize: "0.75rem", letterSpacing: "0.05em" }}>
                        Sedang Diproses (Active AI)
                      </p>
                    </div>
                    <h2 className={`stat-value ${waQueueStats?.total_processing && waQueueStats.total_processing > 0 ? "text-indigo pulse-fast" : "text-muted"}`} style={{ fontSize: "2rem", fontWeight: 800, margin: "0.25rem 0", color: waQueueStats?.total_processing && waQueueStats.total_processing > 0 ? "#4f46e5" : "var(--muted)" }}>
                      {waQueueStats?.total_processing || 0}
                    </h2>
                  </div>
                  <p className="stat-meta" style={{ margin: "0.5rem 0 0", fontSize: "0.82rem", color: "var(--muted)" }}>
                    {waQueueStats?.total_processing && waQueueStats.total_processing > 0 ? (
                      <span style={{ display: "flex", alignItems: "center", gap: "6px" }}>
                        <span className="badge-dot bg-success pulse-fast" style={{ width: "8px", height: "8px", borderRadius: "50%", background: "#10b981", display: "inline-block" }}></span>
                        Panggilan API LLM sedang berjalan...
                      </span>
                    ) : "Tidak ada pemrosesan aktif"}
                  </p>
                </div>

                {/* CARD 4: Rata-rata Latensi AI */}
                <div 
                  className="surface card shadow-soft stat-card"
                  style={{
                    position: "relative",
                    background: "linear-gradient(135deg, var(--surface) 0%, rgba(16, 185, 129, 0.05) 100%)",
                    borderLeft: "4px solid #10b981",
                    transition: "transform 0.2s, box-shadow 0.2s",
                    display: "flex",
                    flexDirection: "column",
                    justifyContent: "space-between",
                    padding: "1.5rem"
                  }}
                  onMouseEnter={e => {
                    e.currentTarget.style.transform = "translateY(-2px)";
                    e.currentTarget.style.boxShadow = "var(--shadow-lg)";
                  }}
                  onMouseLeave={e => {
                    e.currentTarget.style.transform = "none";
                    e.currentTarget.style.boxShadow = "var(--shadow-soft)";
                  }}
                >
                  <div>
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "0.5rem" }}>
                      <p className="stat-label" style={{ fontWeight: 600, color: "var(--muted)", margin: 0, textTransform: "uppercase", fontSize: "0.75rem", letterSpacing: "0.05em" }}>
                        Rata-rata Latensi AI
                      </p>
                    </div>
                    <h2 className="stat-value text-emerald" style={{ fontSize: "2rem", fontWeight: 800, margin: "0.25rem 0", color: "#10b981" }}>
                      {waQueueStats?.average_latency_ms ? `${(waQueueStats.average_latency_ms / 1000).toFixed(2)}s` : "0.00s"}
                    </h2>
                  </div>
                  <p className="stat-meta" style={{ margin: "0.5rem 0 0", fontSize: "0.82rem", color: "var(--muted)" }}>
                    Kecepatan respon rata-rata Chatbot
                  </p>
                </div>
              </div>

              {/* Historical Trend Charts */}
              {waChartData.length > 0 && (
                <div 
                  className="card-grid spacing-mb-lg" 
                  style={{ 
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fit, minmax(320px, 1fr))", 
                    gap: "1.25rem",
                    marginBottom: "2rem" 
                  }}
                >
                  {/* Chart 1: Antrean & Proses AI */}
                  <div className="surface card shadow-soft" style={{ padding: "1.5rem" }}>
                    <h4 style={{ margin: "0 0 1rem 0", fontSize: "0.95rem", fontWeight: 700, textTransform: "uppercase", letterSpacing: "0.05em", color: "var(--muted)" }}>
                      Trend Antrean & Pemrosesan AI
                    </h4>
                    <div ref={queueChartRef} style={{ height: "260px", width: "100%" }} />
                  </div>

                  {/* Chart 2: Status Respon Chatbot AI */}
                  <div className="surface card shadow-soft" style={{ padding: "1.5rem" }}>
                    <h4 style={{ margin: "0 0 1rem 0", fontSize: "0.95rem", fontWeight: 700, textTransform: "uppercase", letterSpacing: "0.05em", color: "var(--muted)" }}>
                      Trend Keberhasilan Respon AI
                    </h4>
                    <div ref={statusChartRef} style={{ height: "260px", width: "100%" }} />
                  </div>

                  {/* Chart 3: Rata-rata Latensi AI */}
                  <div className="surface card shadow-soft" style={{ padding: "1.5rem" }}>
                    <h4 style={{ margin: "0 0 1rem 0", fontSize: "0.95rem", fontWeight: 700, textTransform: "uppercase", letterSpacing: "0.05em", color: "var(--muted)" }}>
                      Rata-rata Latensi Respon AI
                    </h4>
                    <div ref={latencyChartRef} style={{ height: "260px", width: "100%" }} />
                  </div>
                </div>
              )}

              {/* Data Table Card */}
              <div className="surface card shadow-soft">
                <div className="card-header-actions" style={{ display: "flex", flexWrap: "wrap", gap: "16px", justifyContent: "space-between", alignItems: "center", marginBottom: "1.5rem", paddingBottom: "1.5rem", borderBottom: "1px solid rgba(255,255,255,0.06)" }}>
                  <h3 className="form-title" style={{ margin: 0 }}>Log Antrean & Riwayat Chatbot AI</h3>
                  
                  {/* Controls Wrapper */}
                  <div style={{ display: "flex", flexWrap: "wrap", gap: "12px", alignItems: "center" }}>
                    
                    {/* Auto-Refresh Select */}
                    <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                      <span className="text-xs text-muted" style={{ fontWeight: 600, textTransform: "uppercase" }}>Auto-Refresh:</span>
                      <select 
                        className="input-control" 
                        value={waAutoRefresh} 
                        onChange={e => setWaAutoRefresh(e.target.value)}
                        style={{ padding: "6px 12px", fontSize: "0.85rem", height: "38px", minWidth: "120px", background: "rgba(0,0,0,0.2)", color: "var(--foreground)", border: "1px solid rgba(255,255,255,0.1)" }}
                      >
                        <option value="off">Off (Manual)</option>
                        <option value="1m">1 Menit</option>
                        <option value="5m">5 Menit</option>
                      </select>
                    </div>

                    {/* Date Range Select */}
                    <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                      <span className="text-xs text-muted" style={{ fontWeight: 600, textTransform: "uppercase" }}>Filter Tanggal:</span>
                      <select 
                        className="input-control" 
                        value={waDateRange} 
                        onChange={e => setWaDateRange(e.target.value)}
                        style={{ padding: "6px 12px", fontSize: "0.85rem", height: "38px", minWidth: "140px", background: "rgba(0,0,0,0.2)", color: "var(--foreground)", border: "1px solid rgba(255,255,255,0.1)" }}
                      >
                        <option value="all">Semua Waktu</option>
                        <option value="today">Hari Ini</option>
                        <option value="7days">7 Hari Terakhir</option>
                        <option value="30days">30 Hari Terakhir</option>
                        <option value="custom">Custom Range</option>
                      </select>
                    </div>

                    {/* Custom Date Pickers (only shown if waDateRange === "custom") */}
                    {waDateRange === "custom" && (
                      <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                        <input 
                          type="date" 
                          className="input-control" 
                          value={waStartDate} 
                          onChange={e => setWaStartDate(e.target.value)}
                          style={{ padding: "6px 10px", fontSize: "0.85rem", height: "38px", width: "135px", background: "rgba(0,0,0,0.2)", color: "var(--foreground)", border: "1px solid rgba(255,255,255,0.1)" }} 
                        />
                        <span className="text-muted text-xs">s/d</span>
                        <input 
                          type="date" 
                          className="input-control" 
                          value={waEndDate} 
                          onChange={e => setWaEndDate(e.target.value)}
                          style={{ padding: "6px 10px", fontSize: "0.85rem", height: "38px", width: "135px", background: "rgba(0,0,0,0.2)", color: "var(--foreground)", border: "1px solid rgba(255,255,255,0.1)" }} 
                        />
                      </div>
                    )}

                    <div className="search-control" style={{ position: "relative", minWidth: "200px" }}>
                      <input 
                        type="text" 
                        className="input-control" 
                        placeholder="Cari No HP, error..." 
                        value={waQueueSearch}
                        onChange={e => {
                          setWaQueueSearch(e.target.value);
                          setWaQueueOffset(0);
                        }}
                        style={{ paddingRight: "30px", height: "38px" }}
                      />
                      {waQueueSearch && (
                        <button 
                          className="btn-clear" 
                          onClick={() => setWaQueueSearch("")} 
                          style={{ position: "absolute", right: "10px", top: "50%", transform: "translateY(-50%)", border: "none", background: "none", cursor: "pointer", opacity: 0.6 }}
                        >
                          ✕
                        </button>
                      )}
                    </div>
                    <button 
                      className={`btn btn-secondary ${loadingWaQueue ? "loading" : ""}`}
                      onClick={() => {
                        loadWhatsAppQueueStats();
                        loadWhatsAppQueueHistory(waQueueLimit, waQueueOffset, waQueueSearch);
                      }}
                      disabled={loadingWaQueue}
                      style={{ height: "38px" }}
                    >
                      {loadingWaQueue ? "Memuat..." : "Refresh"}
                    </button>
                  </div>
                </div>

                <div className="table-responsive">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>Waktu Diterima</th>
                        <th>WhatsApp & Pengirim</th>
                        <th>Status</th>
                        <th>Latensi</th>
                        <th>Aksi</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(() => {
                        const filteredHistory = (() => {
                          if (waDateRange === "all") return waQueueHistory;
                          const now = new Date();
                          return waQueueHistory.filter(item => {
                            const itemDate = new Date(item.received_at);
                            if (waDateRange === "today") {
                              return itemDate.toDateString() === now.toDateString();
                            }
                            if (waDateRange === "7days") {
                              const diffTime = Math.abs(now.getTime() - itemDate.getTime());
                              const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
                              return diffDays <= 7;
                            }
                            if (waDateRange === "30days") {
                              const diffTime = Math.abs(now.getTime() - itemDate.getTime());
                              const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
                              return diffDays <= 30;
                            }
                            if (waDateRange === "custom") {
                              if (waStartDate) {
                                const start = new Date(waStartDate);
                                start.setHours(0,0,0,0);
                                if (itemDate < start) return false;
                              }
                              if (waEndDate) {
                                const end = new Date(waEndDate);
                                end.setHours(23,59,59,999);
                                if (itemDate > end) return false;
                              }
                              return true;
                            }
                            return true;
                          });
                        })();

                        if (loadingWaQueue && filteredHistory.length === 0) {
                          return (
                            <tr>
                              <td colSpan={5} className="text-center-muted" style={{ padding: "3rem" }}>
                                <div className="spinner spacing-mb-sm" style={{ margin: "0 auto" }}></div>
                                Memuat data antrean...
                              </td>
                            </tr>
                          );
                        }

                        if (filteredHistory.length === 0) {
                          return (
                            <tr>
                              <td colSpan={5} className="text-center-muted" style={{ padding: "3rem" }}>
                                Tidak ada log pesan untuk filter yang dipilih.
                              </td>
                            </tr>
                          );
                        }

                        return filteredHistory.map(item => {
                          const statusClass = 
                            item.status === "success" ? "running" :
                            item.status === "failed" ? "stopped" :
                            item.status === "processing" ? "pending" : "idle";

                          const statusLabel = 
                            item.status === "success" ? "SUKSES" :
                            item.status === "failed" ? "GAGAL" :
                            item.status === "processing" ? "DIPROSES" : "ANTRE";

                          return (
                            <tr key={item.id}>
                              <td style={{ whiteSpace: "nowrap" }}>
                                <span className="text-xs opacity-80">{new Date(item.received_at).toLocaleString("id-ID")}</span>
                              </td>
                              <td>
                                <div style={{ display: "flex", flexDirection: "column" }}>
                                  <strong className="text-indigo">{item.phone_number}</strong>
                                  {item.tenant_code ? (
                                    <span className="text-xs font-mono text-muted" style={{ marginTop: "2px" }}>
                                      Workspace: <strong className="text-emerald">{item.tenant_code}</strong>
                                    </span>
                                  ) : (
                                    <span className="text-xs text-muted" style={{ marginTop: "2px" }}>Global/Unknown</span>
                                  )}
                                </div>
                              </td>
                              <td>
                                <div style={{ display: "flex", flexDirection: "column", gap: "4px", alignItems: "flex-start" }}>
                                  <span className={`badge-status ${statusClass}`}>{statusLabel}</span>
                                  {item.status === "failed" && item.error_message && (
                                    <div className="text-rose text-xs" style={{ background: "rgba(239, 68, 68, 0.08)", padding: "6px 10px", borderRadius: "6px", borderLeft: "3px solid #ef4444", marginTop: "4px", maxWidth: "450px", wordBreak: "break-word", whiteSpace: "normal" }}>
                                      <strong>Error:</strong> {item.error_message}
                                    </div>
                                  )}
                                </div>
                              </td>
                              <td>
                                {item.processing_time_ms ? (
                                  <span className="text-sm font-mono">{(item.processing_time_ms / 1000).toFixed(2)}s</span>
                                ) : (
                                  <span className="text-muted">-</span>
                                )}
                              </td>
                              <td>
                                <div style={{ display: "flex", gap: "6px" }}>
                                  {item.status === "failed" && (
                                    <button 
                                      className="btn btn-secondary btn-sm"
                                      onClick={() => handleRetryWhatsAppMessage(item.id)}
                                      disabled={retryingMsgId === item.id}
                                      style={{ padding: "4px 8px", fontSize: "0.75rem" }}
                                    >
                                      {retryingMsgId === item.id ? "Retrying..." : "Retry"}
                                    </button>
                                  )}
                                </div>
                              </td>
                            </tr>
                          );
                        });
                      })()}
                    </tbody>
                  </table>
                </div>

                {/* Pagination */}
                {waQueueTotal > waQueueLimit && (
                  <div className="pagination-bar" style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginTop: "1.5rem", borderTop: "1px solid rgba(255,255,255,0.08)", paddingTop: "1rem" }}>
                    <span className="text-xs text-muted">
                      Menampilkan <strong>{waQueueOffset + 1}</strong> - <strong>{Math.min(waQueueOffset + waQueueLimit, waQueueTotal)}</strong> dari <strong>{waQueueTotal}</strong> log
                    </span>
                    <div style={{ display: "flex", gap: "6px" }}>
                      <button 
                        className="btn btn-secondary btn-sm"
                        disabled={waQueueOffset === 0 || loadingWaQueue}
                        onClick={() => setWaQueueOffset(Math.max(0, waQueueOffset - waQueueLimit))}
                      >
                        Sebelumnya
                      </button>
                      <button 
                        className="btn btn-secondary btn-sm"
                        disabled={waQueueOffset + waQueueLimit >= waQueueTotal || loadingWaQueue}
                        onClick={() => setWaQueueOffset(waQueueOffset + waQueueLimit)}
                      >
                        Selanjutnya
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTab === "storage" && (
            <div className="card-grid tight">
               <div className="surface card shadow-soft" style={{ gridColumn: "1 / -1" }}>
                 <h3 className="form-title">Pilih Provider Storage Utama</h3>
                 <p className="form-section-desc">Pilih di mana file lampiran, struk, dan backup akan disimpan.</p>
                 <div style={{ display: "flex", gap: "10px", alignItems: "center", marginTop: "1rem" }}>
                   <select className="input-control" value={storageActiveProvider} onChange={e => setStorageActiveProvider(e.target.value)} style={{ maxWidth: "300px" }}>
                     <option value="local">Local Filesystem / NFS</option>
                     <option value="s3">Amazon S3 / DigitalOcean Spaces</option>
                     <option value="gdrive">Google Drive</option>
                   </select>
                   <button className="btn btn-primary" onClick={() => handleSaveStorageConfig("active")}>Simpan Pilihan</button>
                 </div>
               </div>

               <div className="surface card shadow-soft">
                 <div className="form-header-with-badge">
                   <h3 className="form-title">Local / NFS Storage</h3>
                   {storageActiveProvider === "local" && <span className="badge-status running">AKTIF</span>}
                 </div>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveStorageConfig("local"); }}>
                   <label className="form-field">Path Penyimpanan <input className="input-control" placeholder="./data/storage" value={localConfig.path} onChange={e => setLocalConfig({path: e.target.value})} /></label>
                   <p className="opacity-60 text-xs">Pastikan direktori ini memiliki izin tulis (writable) oleh aplikasi.</p>
                   <button className="btn btn-primary spacing-mt-md" type="submit">Simpan Konfigurasi</button>
                 </form>
               </div>

               <div className="surface card shadow-soft">
                 <div className="form-header-with-badge">
                   <h3 className="form-title">Amazon S3 / Compatible</h3>
                   {storageActiveProvider === "s3" && <span className="badge-status running">AKTIF</span>}
                 </div>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveStorageConfig("s3"); }}>
                   <div className="card-grid two-col" style={{ gap: "10px" }}>
                     <label className="form-field">Region <input className="input-control" placeholder="us-east-1" value={s3Config.region} onChange={e => setS3Config({...s3Config, region: e.target.value})} /></label>
                     <label className="form-field">Bucket <input className="input-control" placeholder="my-bucket" value={s3Config.bucket} onChange={e => setS3Config({...s3Config, bucket: e.target.value})} /></label>
                   </div>
                   <label className="form-field">Access Key <input className="input-control" placeholder="AKIA..." value={s3Config.accessKey} onChange={e => setS3Config({...s3Config, accessKey: e.target.value})} /></label>
                   <label className="form-field">Secret Key <PasswordInput placeholder={s3KeySaved ? "•••••••• (Sudah tersimpan)" : "Secret Key"} value={s3Config.secretKey} onChange={e => setS3Config({...s3Config, secretKey: e.target.value})} /></label>
                   <label className="form-field">Custom Endpoint (Optional) <input className="input-control" placeholder="https://s3.digitaloceanspaces.com" value={s3Config.endpoint} onChange={e => setS3Config({...s3Config, endpoint: e.target.value})} /></label>
                   <button className="btn btn-primary spacing-mt-md" type="submit">Simpan Konfigurasi</button>
                 </form>
               </div>

               <div className="surface card shadow-soft">
                 <div className="form-header-with-badge">
                   <h3 className="form-title">Google Drive</h3>
                   {storageActiveProvider === "gdrive" && <span className="badge-status running">AKTIF</span>}
                 </div>
                 <form className="form-grid" onSubmit={(e) => { e.preventDefault(); handleSaveStorageConfig("gdrive"); }}>
                   <label className="form-field">Folder ID <input className="input-control" placeholder="1abc..." value={gdriveConfig.folderId} onChange={e => setGdriveConfig({...gdriveConfig, folderId: e.target.value})} /></label>
                   <label className="form-field">Service Account JSON <textarea className="input-control" style={{ height: "100px", fontFamily: "monospace", fontSize: "12px" }} placeholder='{"type": "service_account", ...}' value={gdriveConfig.credentialsJson} onChange={e => setGdriveConfig({...gdriveConfig, credentialsJson: e.target.value})} /></label>
                   <button className="btn btn-primary spacing-mt-md" type="submit">Simpan Konfigurasi</button>
                 </form>
               </div>
            </div>
          )}

          {activeTab === "updates" && (
            <div className="updates-view" style={{ display: "flex", flexDirection: "column", gap: "2rem" }}>
              <div className="surface card shadow-soft" style={{ padding: "2rem" }}>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: "1rem" }}>
                  <div>
                    <h3 className="form-title" style={{ margin: 0, fontSize: "1.25rem", fontWeight: 700 }}>
                      Status Sinkronisasi GitHub
                    </h3>
                    <p className="opacity-70 text-sm" style={{ marginTop: "4px" }}>
                      Hubungkan server lokal dengan repositori Pekan untuk deployment otomatis sekali klik.
                    </p>
                  </div>
                  <div style={{ display: "flex", gap: "10px" }}>
                    <button 
                      className="btn btn-secondary-outline" 
                      onClick={handleCheckUpdate} 
                      disabled={checkingUpdate || applyingUpdate}
                      style={{ display: "flex", alignItems: "center", gap: "8px" }}
                    >
                      {checkingUpdate ? (
                        <span className="spinner spinner-xs"></span>
                      ) : (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/></svg>
                      )}
                      Periksa Pembaruan
                    </button>
                    {updateInfo?.update_available && (
                      <button 
                        className="btn btn-primary" 
                        onClick={handleApplyUpdate} 
                        disabled={applyingUpdate}
                        style={{ display: "flex", alignItems: "center", gap: "8px", background: "linear-gradient(135deg, var(--primary) 0%, #4f46e5 100%)", border: "none" }}
                      >
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"/></svg>
                        Perbarui Sekarang
                      </button>
                    )}
                  </div>
                </div>

                <div className="spacing-mt-lg" style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))", gap: "1.5rem" }}>
                  <div className="surface card shadow-softstat-card" style={{ background: "rgba(0,0,0,0.02)", border: "1px solid rgba(0,0,0,0.05)", padding: "1.5rem", borderRadius: "12px" }}>
                    <span className="text-xs text-muted" style={{ fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.05em" }}>Versi Saat Ini (Local)</span>
                    {updateInfo ? (
                      <div className="spacing-mt-sm">
                        <strong className="text-lg font-mono text-indigo" style={{ display: "block" }}>
                          {updateInfo.is_git_repo ? updateInfo.current_commit.slice(0, 8) : "Bukan repositori Git"}
                        </strong>
                        <span className="text-xs text-muted" style={{ display: "block", marginTop: "4px" }}>
                          {updateInfo.is_git_repo ? new Date(updateInfo.current_date).toLocaleString("id-ID") : "Instalasi manual (Zip)"}
                        </span>
                      </div>
                    ) : (
                      <div className="spacing-mt-sm opacity-50">Menghubungkan ke git...</div>
                    )}
                  </div>

                  <div className="surface card shadow-softstat-card" style={{ background: "rgba(0,0,0,0.02)", border: "1px solid rgba(0,0,0,0.05)", padding: "1.5rem", borderRadius: "12px" }}>
                    <span className="text-xs text-muted" style={{ fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.05em" }}>Versi Terbaru (GitHub Remote)</span>
                    {updateInfo ? (
                      <div className="spacing-mt-sm">
                        <strong className="text-lg font-mono text-emerald" style={{ display: "block" }}>
                          {updateInfo.latest_commit ? updateInfo.latest_commit.slice(0, 8) : "Tidak terhubung"}
                        </strong>
                        <span className="text-xs text-muted" style={{ display: "block", marginTop: "4px", whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
                          {updateInfo.latest_message || "Tidak ada pesan commit"}
                        </span>
                      </div>
                    ) : (
                      <div className="spacing-mt-sm opacity-50">Mengambil data dari remote...</div>
                    )}
                  </div>

                  <div className="surface card shadow-softstat-card" style={{ background: "rgba(0,0,0,0.02)", border: "1px solid rgba(0,0,0,0.05)", padding: "1.5rem", borderRadius: "12px", display: "flex", flexDirection: "column", justifyContent: "center" }}>
                    <span className="text-xs text-muted" style={{ fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.05em" }}>Status Pembaruan</span>
                    <div className="spacing-mt-sm" style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                      {updateInfo?.update_available ? (
                        <>
                          <span className="badge-status stopped" style={{ background: "#fef2f2", color: "#ef4444", fontSize: "0.85rem", padding: "4px 12px" }}>
                            UPDATE TERSEDIA
                          </span>
                          <span className="text-xs text-rose" style={{ fontWeight: 600 }}>Diperlukan sinkronisasi</span>
                        </>
                      ) : updateInfo ? (
                        <>
                          <span className="badge-status running" style={{ background: "#ecfdf5", color: "#10b981", fontSize: "0.85rem", padding: "4px 12px" }}>
                            UP TO DATE
                          </span>
                          <span className="text-xs text-emerald" style={{ fontWeight: 600 }}>Sistem sudah optimal</span>
                        </>
                      ) : (
                        <span className="text-xs opacity-50">Menunggu pemeriksaan...</span>
                      )}
                    </div>
                  </div>
                </div>
              </div>

              <div className="surface card shadow-soft" style={{ background: "#0f172a", border: "1px solid #1e293b", padding: "1.5rem", borderRadius: "12px", display: "flex", flexDirection: "column", minHeight: "450px" }}>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", borderBottom: "1px solid #1e293b", paddingBottom: "1rem", marginBottom: "1rem" }}>
                  <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                    <div style={{ display: "flex", gap: "6px" }}>
                      <span style={{ width: "12px", height: "12px", borderRadius: "50%", background: "#ef4444" }}></span>
                      <span style={{ width: "12px", height: "12px", borderRadius: "50%", background: "#f59e0b" }}></span>
                      <span style={{ width: "12px", height: "12px", borderRadius: "50%", background: "#10b981" }}></span>
                    </div>
                    <span className="font-mono text-xs" style={{ color: "#94a3b8", marginLeft: "8px", fontWeight: 600 }}>
                      pekan-deployer@bash ~ stdout console log
                    </span>
                  </div>
                  <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                    {updateProgress?.status === "running" && (
                      <span className="text-xs font-mono text-amber" style={{ animation: "pulse 1.5s infinite" }}>
                        ● SEDANG MEMPROSES DEPLOYMENT...
                      </span>
                    )}
                    {updateProgress?.status === "success" && (
                      <span className="text-xs font-mono text-emerald">
                        ● DEPLOYMENT BERHASIL! (Reloading...)
                      </span>
                    )}
                    {updateProgress?.status === "failed" && (
                      <span className="text-xs font-mono text-rose">
                        ● DEPLOYMENT GAGAL: {updateProgress.error}
                      </span>
                    )}
                  </div>
                </div>

                <div 
                  style={{ 
                    flex: 1, 
                    background: "#020617", 
                    borderRadius: "8px", 
                    padding: "1.5rem", 
                    overflowY: "auto", 
                    maxHeight: "350px", 
                    fontFamily: "'Courier New', Courier, monospace", 
                    fontSize: "0.85rem", 
                    lineHeight: "1.6", 
                    color: "#38bdf8", 
                    whiteSpace: "pre-wrap", 
                    boxShadow: "inset 0 2px 8px rgba(0,0,0,0.8)" 
                  }}
                >
                  {updateProgress?.logs ? (
                    updateProgress.logs
                  ) : (
                    <div style={{ color: "#64748b", fontStyle: "italic" }}>
                      Belum ada aktivitas deployment. Klik "Perbarui Sekarang" untuk memulai build ulang infrastruktur.
                    </div>
                  )}
                  <div ref={terminalLogEndRef} />
                </div>
              </div>
            </div>
          )}
        </main>
      </div>

      <DeleteConfirmModal
        isOpen={!!tenantToDelete}
        title="Hapus Workspace"
        message={`Hapus workspace "${tenantToDelete?.name}"? Semua data infrastruktur, user, dan transaksi terkait akan dihapus secara permanen.`}
        isLoading={loading}
        onConfirm={() => tenantToDelete && handleDeleteTenant(tenantToDelete.id)}
        onCancel={() => setTenantToDelete(null)}
      />

      <DeleteConfirmModal
        isOpen={!!fileToRestore}
        title="Peringatan Restore Database!"
        message={`Anda akan me-restore database menggunakan file "${fileToRestore}". Peringatan: Proses ini (Clean Restore) akan MENGHAPUS SEMUA DATA yang ada saat ini dan menggantinya dengan data dari backup. Lanjutkan?`}
        isLoading={isRestoring}
        onConfirm={handleRestoreBackup}
        onCancel={() => setFileToRestore(null)}
      />
      <ToastContainer toasts={toasts} onRemove={remove} />
      <BackToTop />

      {userToReset && (
        <div className="modal-overlay" style={{ zIndex: 2000 }}>
          <div className="surface card shadow-strong modal-sm">
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "start", marginBottom: "0.5rem" }}>
              <h3 className="form-title" style={{ margin: 0 }}>
                {userToReset.action === 'password' ? 'Reset Password' : 
                 userToReset.action === 'email' ? 'Ubah Email' : 'Ubah Nomor HP'}
              </h3>
              {userToReset.action === 'password' && (
                <button 
                  type="button"
                  className="btn btn-ghost-inline btn-sm" 
                  onClick={() => {
                    const upper = "ABCDEFGHJKLMNPQRSTUVWXYZ";
                    const lower = "abcdefghijkmnopqrstuvwxyz";
                    const nums = "23456789";
                    const syms = "!@#$%^&*";
                    
                    // Start with a prefix and ensure at least one of each required type
                    let pass = "Pkn-";
                    pass += upper.charAt(Math.floor(Math.random() * upper.length));
                    pass += lower.charAt(Math.floor(Math.random() * lower.length));
                    pass += nums.charAt(Math.floor(Math.random() * nums.length));
                    pass += syms.charAt(Math.floor(Math.random() * syms.length));
                    
                    // Fill the rest to reach 12+ chars
                    const all = upper + lower + nums + syms;
                    for(let i=0; i<6; i++) pass += all.charAt(Math.floor(Math.random() * all.length));
                    
                    setResetValue(pass);
                  }}
                  style={{ color: "var(--primary)", fontWeight: 600 }}
                >
                  Buat Password Acak
                </button>
              )}
            </div>
            <p className="spacing-mb-md opacity-70">
              Update {userToReset.action === 'password' ? 'password' : userToReset.action} untuk user: <strong>{userToReset.user.full_name}</strong>
              {userToReset.action === 'password' && <><br/><span className="text-xs text-primary">User akan diwajibkan ganti password saat login pertama kali.</span></>}
            </p>
            <form onSubmit={(e) => { 
              e.preventDefault(); 
              e.stopPropagation();
              if(!resetValue) return;
              handleUserReset(); 
            }} className="form-grid">
              <label className="form-field">
                {userToReset.action === 'password' ? 'Password Baru' : 
                 userToReset.action === 'email' ? 'Email Baru' : 'Nomor HP Baru'}
                <div style={{ position: "relative" }}>
                  <input 
                    className="input-control" 
                    type={userToReset.action === 'password' ? 'text' : 'text'}
                    placeholder={userToReset.action === 'password' ? 'Contoh: PEKAN-X7B2' : 'Masukkan nilai baru'} 
                    value={resetValue}
                    onChange={e => setResetValue(e.target.value)}
                    autoFocus
                    autoComplete="new-password"
                    style={userToReset.action === 'password' ? { fontFamily: "monospace", letterSpacing: "1px", fontWeight: 700 } : {}}
                  />
                </div>
              </label>
              <div className="form-actions spacing-mt-md" style={{ display: "flex", gap: "10px", justifyContent: "flex-end", flexWrap: "wrap" }}>
                {userToReset.action === 'password' && (
                  <button 
                    className="btn btn-secondary-outline" 
                    type="button" 
                    style={{ marginRight: "auto" }}
                    onClick={() => {
                      if (!editingTenant) return;
                      const url = `${window.location.origin}/reset-password?t=${editingTenant.code}&e=${encodeURIComponent(userToReset.user.email)}`;
                      navigator.clipboard.writeText(url);
                      success("Link reset password berhasil disalin ke clipboard.");
                    }}
                  >
                    Salin Link Reset
                  </button>
                )}
                <button className="btn btn-secondary-outline" type="button" onClick={() => { setUserToReset(null); setResetValue(""); }}>Batal</button>
                <button className="btn btn-primary" type="submit" disabled={isResetting || !resetValue}>
                  {isResetting ? "Memproses..." : "Simpan Perubahan"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showTestModal && (
        <div className="modal-overlay">
          <div className="surface card shadow-strong modal-sm">
            <h3 className="form-title">Test Connection: {testProvider.toUpperCase()}</h3>
            <p className="spacing-mb-md opacity-70">
              Masukkan {testProvider === "smtp" ? "Email" : "Nomor WhatsApp (dengan kode negara, misal: 62812...)"} tujuan untuk mengirim pesan uji coba.
            </p>
            <div className="form-grid">
              <label className="form-field">
                Tujuan Pengiriman
                <input 
                  className="input-control" 
                  placeholder={testProvider === "smtp" ? "email@contoh.com" : "62812345678"} 
                  value={testDestination}
                  onChange={e => setTestDestination(e.target.value)}
                  autoFocus
                />
              </label>
              <div className="form-actions spacing-mt-md" style={{ display: "flex", gap: "10px", justifyContent: "flex-end" }}>
                <button className="btn btn-secondary-outline" onClick={() => setShowTestModal(false)}>Batal</button>
                <button className="btn btn-primary" onClick={confirmTestConnection}>Kirim Test</button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
