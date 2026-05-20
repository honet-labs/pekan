import { Navigate, Route, Routes } from "react-router-dom";
import { AppShell } from "../layout/AppShell";
import { FeatureGuard } from "../guards/FeatureGuard";
import { PermissionGuard } from "../guards/PermissionGuard";
import { ProtectedRoute } from "../guards/ProtectedRoute";
import { TenantRoute } from "../guards/TenantRoute";
import { FinanceMasterDataPage } from "../../features/finance/masterdata/pages/FinanceMasterDataPage";
import { TransactionCreatePage } from "../../features/finance/transactions/pages/TransactionCreatePage";
import { TransactionDetailPage } from "../../features/finance/transactions/pages/TransactionDetailPage";
import { TransactionListPage } from "../../features/finance/transactions/pages/TransactionListPage";
import { LoginPage } from "../../features/core/auth/pages/LoginPage";
import { ForgotPasswordPage } from "../../features/core/auth/pages/ForgotPasswordPage";
import { ForgotTenantPage } from "../../features/core/auth/pages/ForgotTenantPage";
import { ResetPasswordPage } from "../../features/core/auth/pages/ResetPasswordPage";
import { FinanceDashboardPage } from "../../features/finance/dashboard/pages/FinanceDashboardPage";
import { SavingsListPage } from "../../features/finance/savings/pages/SavingsListPage";
import { SavingsCreatePage } from "../../features/finance/savings/pages/SavingsCreatePage";
import { SavingsDetailPage } from "../../features/finance/savings/pages/SavingsDetailPage";
import { BudgetsListPage } from "../../features/finance/budgets/pages/BudgetsListPage";
import { BudgetCreatePage } from "../../features/finance/budgets/pages/BudgetCreatePage";
import { BudgetDetailPage } from "../../features/finance/budgets/pages/BudgetDetailPage";
import { ReminderDetailPage } from "../../features/finance/reminders/pages/ReminderDetailPage";
import { ReminderCreatePage } from "../../features/finance/reminders/pages/ReminderCreatePage";
import { RemindersPage } from "../../features/finance/reminders/pages/RemindersPage";
import { ReportsPage } from "../../features/finance/reports/pages/ReportsPage";
import { NotificationsPage } from "../../features/finance/notifications/pages/NotificationsPage";
import { SettingsPage } from "../../features/finance/settings/pages/SettingsPage";
import { SettingsTemplatesPage } from "../../features/finance/settings/pages/SettingsTemplatesPage";
import { SettingsUsersPage } from "../../features/finance/settings/pages/SettingsUsersPage";
import { SettingsWhatsAppPage } from "../../features/finance/settings/pages/SettingsWhatsAppPage";
import { SettingsCategoriesPage } from "../../features/finance/settings/pages/SettingsCategoriesPage";
import { AuditLogsPage } from "../../features/finance/settings/pages/AuditLogsPage";
import { ProfilePage } from "../../features/core/profile/pages/ProfilePage";
import { ReceiptScanPage } from "../../features/finance/receipts/pages/ReceiptScanPage";
import { SettingsReceiptScanPage } from "../../features/finance/settings/pages/SettingsReceiptScanPage";
import { AdminDashboardPage } from "../../features/core/admin/pages/AdminDashboardPage";
import { ChatbotPage } from "../../features/finance/chatbot/pages/ChatbotPage";
import NotFoundPage from "../pages/NotFoundPage";
import { useTenantStore } from "../../core/tenant/tenant-store";

export function AppRouter(): JSX.Element {
  const tenant = useTenantStore();
  const activeID = tenant.activeTenantID || "default";

  return (
    <Routes>
      <Route path="/" element={<Navigate to={`/app/${activeID}/finance/dashboard`} replace />} />
      <Route path="/app" element={<Navigate to="/login" replace />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="/forgot-password" element={<ForgotPasswordPage />} />
      <Route path="/forgot-tenant" element={<ForgotTenantPage />} />
      <Route path="/reset-password" element={<ResetPasswordPage />} />
      <Route path="/admin" element={<AdminDashboardPage />} />
      <Route
        path="/app/:tenantCode/*"
        element={
          <ProtectedRoute>
            <TenantRoute>
              <AppShell />
            </TenantRoute>
          </ProtectedRoute>
        }
      >
        <Route index element={<Navigate to="finance/dashboard" replace />} />
        <Route
          path="finance/dashboard"
          element={
            <FeatureGuard feature="finance.dashboard.read">
              <PermissionGuard permission="finance.dashboard.read">
                <FinanceDashboardPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/transactions"
          element={
            <FeatureGuard feature="finance.transactions.read">
              <TransactionListPage />
            </FeatureGuard>
          }
        />
        <Route
          path="finance/transactions/new"
          element={
            <FeatureGuard feature="finance.transactions.write">
              <PermissionGuard permission="finance.transactions.create">
                <TransactionCreatePage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/transactions/:transactionID"
          element={
            <FeatureGuard feature="finance.transactions.read">
              <TransactionDetailPage />
            </FeatureGuard>
          }
        />
        <Route
          path="finance/savings"
          element={
            <FeatureGuard feature="finance.savings.read">
              <PermissionGuard permission="finance.savings.read">
                <SavingsListPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/savings/new"
          element={
            <FeatureGuard feature="finance.savings.write">
              <PermissionGuard permission="finance.savings.create">
                <SavingsCreatePage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/savings/:savingsID"
          element={
            <FeatureGuard feature="finance.savings.read">
              <PermissionGuard permission="finance.savings.update">
                <SavingsDetailPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/budgets"
          element={
            <FeatureGuard feature="finance.budgets.read">
              <PermissionGuard permission="finance.budgets.read">
                <BudgetsListPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/budgets/new"
          element={
            <FeatureGuard feature="finance.budgets.write">
              <PermissionGuard permission="finance.budgets.create">
                <BudgetCreatePage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/budgets/:budgetID"
          element={
            <FeatureGuard feature="finance.budgets.read">
              <PermissionGuard permission="finance.budgets.update">
                <BudgetDetailPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/reminders"
          element={
            <FeatureGuard feature="finance.reminders.read">
              <PermissionGuard permission="finance.reminders.read">
                <RemindersPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/reminders/new"
          element={
            <FeatureGuard feature="finance.reminders.read">
              <PermissionGuard permission="finance.reminders.create">
                <ReminderCreatePage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/reminders/:reminderID"
          element={
            <FeatureGuard feature="finance.reminders.read">
              <PermissionGuard permission="finance.reminders.read">
                <ReminderDetailPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/master-data"
          element={
            <FeatureGuard feature="finance.masterdata.read">
              <FinanceMasterDataPage />
            </FeatureGuard>
          }
        />
        <Route
          path="finance/receipt-scan"
          element={
            <FeatureGuard feature="finance.transactions.write">
              <PermissionGuard permission="finance.transactions.create">
                <ReceiptScanPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/reports"
          element={
            <FeatureGuard feature="finance.reports.read">
              <PermissionGuard permission="finance.reports.read">
                <ReportsPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/chatbot"
          element={
            <FeatureGuard feature="finance.dashboard.read">
              <PermissionGuard permission="finance.dashboard.read">
                <ChatbotPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/notifications"
          element={
            <FeatureGuard feature="finance.notifications.read">
              <PermissionGuard permission="finance.notifications.read">
                <NotificationsPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route path="finance/settings" element={<Navigate to="notifications" replace />} />
        <Route
          path="finance/settings/notifications"
          element={
            <FeatureGuard feature="finance.settings.read">
              <PermissionGuard permission="finance.settings.read">
                <SettingsPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/settings/templates"
          element={
            <FeatureGuard feature="finance.settings.read">
              <PermissionGuard permission="finance.settings.read">
                <SettingsTemplatesPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/settings/categories"
          element={
            <FeatureGuard feature="finance.masterdata.read">
              <PermissionGuard permission="finance.categories.read">
                <SettingsCategoriesPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/settings/receipt-scan"
          element={
            <FeatureGuard feature="finance.settings.read">
              <PermissionGuard permission="finance.settings.read">
                <SettingsReceiptScanPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/settings/users"
          element={
            <FeatureGuard feature="finance.settings.write">
              <PermissionGuard permission="finance.settings.roles.manage">
                <SettingsUsersPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/settings/whatsapp"
          element={
            <FeatureGuard feature="finance.settings.read">
              <PermissionGuard permission="finance.settings.read">
                <SettingsWhatsAppPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route
          path="finance/logs"
          element={
            <FeatureGuard feature="finance.settings.read">
              <PermissionGuard permission="finance.settings.audit.read">
                <AuditLogsPage />
              </PermissionGuard>
            </FeatureGuard>
          }
        />
        <Route path="profile" element={<ProfilePage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
}
