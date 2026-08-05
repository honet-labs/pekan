import { useEffect, useMemo, useState } from "react";
import {
  createRole,
  createTenantUser,
  deleteRole,
  deleteTenantUser,
  listRoleCatalog,
  listTenantUsers,
  updateRole,
  updateTenantUser
} from "../api/settings.api";
import { TenantPermission, TenantRole, TenantUserRoleEntry } from "../api/settings.types";
import { useI18n } from "../../../../core/i18n/i18n";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";
import { PasswordStrength } from "../../../../core/components/PasswordStrength";
import { PasswordInput } from "../../../../core/components/PasswordInput";
import { PageHeader } from "../../../../core/components/PageHeader";


const defaultRoleForm = {
  code: "",
  name: "",
  permission_ids: [] as string[]
};

const defaultUserForm = {
  email: "",
  full_name: "",
  username: "",
  phone_number: "",
  password: "",
  status: "active",
  is_active: true,
  role_ids: [] as string[]
};

export function SettingsUsersPage(): JSX.Element {
  const { locale, t } = useI18n();
  const { toasts, success, error: showError, remove: removeToast } = useToast();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [roles, setRoles] = useState<TenantRole[]>([]);
  const [permissions, setPermissions] = useState<TenantPermission[]>([]);
  const [users, setUsers] = useState<TenantUserRoleEntry[]>([]);

  const [roleForm, setRoleForm] = useState(defaultRoleForm);
  const [editingRoleID, setEditingRoleID] = useState<string | null>(null);
  const [savingRole, setSavingRole] = useState(false);
  const [deletingRoleID, setDeletingRoleID] = useState<string | null>(null);

  const [userForm, setUserForm] = useState(defaultUserForm);
  const [editingMembershipID, setEditingMembershipID] = useState<string | null>(null);
  const [savingUser, setSavingUser] = useState(false);
  const [deletingMembershipID, setDeletingMembershipID] = useState<string | null>(null);
  const [roleToDelete, setRoleToDelete] = useState<TenantRole | null>(null);
  const [userToDelete, setUserToDelete] = useState<TenantUserRoleEntry | null>(null);

  const groupedPermissions = useMemo(() => {
    const groups = new Map<string, TenantPermission[]>();
    for (const permission of permissions) {
      const list = groups.get(permission.module_code) ?? [];
      list.push(permission);
      groups.set(permission.module_code, list);
    }
    return Array.from(groups.entries()).map(([moduleCode, items]) => ({ moduleCode, items }));
  }, [permissions]);


  const formatPermissionGroup = (moduleCode: string): string => {
    const labels: Record<string, { en: string; id: string }> = {
      "core.entitlement": { en: "Plan & access", id: "Paket & akses" },
      finance: { en: "Finance", id: "Keuangan" },
      "finance.masterdata": { en: "Master data", id: "Master data" },
      "finance.settings": { en: "Settings", id: "Pengaturan" },
      "finance.notifications": { en: "Notifications", id: "Notifikasi" }
    };
    const found = labels[moduleCode];
    if (found) {
      return locale === "id" ? found.id : found.en;
    }
    return moduleCode;
  };

  const formatPermissionLabel = (permission: TenantPermission): string => {
    const special: Record<string, { en: string; id: string }> = {
      "finance.reminders.read": { en: "Read reminders menu", id: "Baca menu pengingat" },
      "finance.reminders.create": { en: "Create reminder", id: "Buat pengingat" },
      "finance.reminders.update": { en: "Update reminder", id: "Ubah pengingat" },
      "finance.reminders.delete": { en: "Delete reminder", id: "Hapus pengingat" },
      "finance.transactions.read": { en: "Read transactions menu", id: "Baca menu transaksi" },
      "finance.transactions.create": { en: "Create transaction", id: "Buat transaksi" },
      "finance.transactions.update": { en: "Update transaction", id: "Ubah transaksi" },
      "finance.transactions.delete": { en: "Delete transaction", id: "Hapus transaksi" },
      "finance.savings.read": { en: "Read savings menu", id: "Baca menu tabungan" },
      "finance.savings.create": { en: "Create savings goal", id: "Buat target tabungan" },
      "finance.savings.update": { en: "Update savings goal", id: "Ubah target tabungan" },
      "finance.savings.delete": { en: "Delete savings goal", id: "Hapus target tabungan" },
      "finance.budgets.read": { en: "Read budgets menu", id: "Baca menu anggaran" },
      "finance.budgets.create": { en: "Create budget", id: "Buat anggaran" },
      "finance.budgets.update": { en: "Update budget", id: "Ubah anggaran" },
      "finance.budgets.delete": { en: "Delete budget", id: "Hapus anggaran" },
      "finance.reports.read": { en: "Read reports menu", id: "Baca menu laporan" },
      "finance.categories.read": { en: "Read categories", id: "Baca kategori" },
      "finance.categories.create": { en: "Create category", id: "Buat kategori" },
      "finance.accounts.read": { en: "Read accounts", id: "Baca akun" },
      "finance.accounts.create": { en: "Create account", id: "Buat akun" },
      "finance.settings.read": { en: "Read settings menu", id: "Baca menu pengaturan" },
      "finance.settings.roles.manage": { en: "Manage roles & users", id: "Kelola role & user" },
      "finance.settings.audit.read": { en: "Read activity logs", id: "Baca log aktivitas" }
    };
    const found = special[permission.code];
    if (found) {
      return locale === "id" ? found.id : found.en;
    }
    if (permission.name?.trim()) {
      return permission.name;
    }
    return permission.code;
  };

  async function loadAll(): Promise<void> {
    setLoading(true);
    setError(null);
    try {
      const [catalog, tenantUsers] = await Promise.all([listRoleCatalog(), listTenantUsers()]);
      setRoles(catalog.roles);
      setPermissions(catalog.permissions);
      setUsers(tenantUsers);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadAll().catch(() => undefined);
  }, [t]);

  async function handleSaveRole(): Promise<void> {
    setSavingRole(true);
    setError(null);
    try {
      if (editingRoleID) {
        await updateRole(editingRoleID, roleForm);
      } else {
        await createRole(roleForm);
      }
      setRoleForm(defaultRoleForm);
      setEditingRoleID(null);
      await loadAll();
      success(editingRoleID ? t("common.updateSuccess") : t("common.saveSuccess"));
    } catch (err) {
      const message = err instanceof Error ? err.message : t("errors.saveDataFailed");
      setError(message);
      showError(message);
    } finally {
      setSavingRole(false);
    }
  }

  function handleEditRole(role: TenantRole): void {
    setEditingRoleID(role.id);
    setRoleForm({
      code: role.code,
      name: role.name,
      permission_ids: role.permission_ids ?? []
    });
  }

  async function handleDeleteRole(roleID: string): Promise<void> {
    setRoleToDelete(null);
    setDeletingRoleID(roleID);
    setError(null);
    try {
      await deleteRole(roleID);
      if (editingRoleID === roleID) {
        setEditingRoleID(null);
        setRoleForm(defaultRoleForm);
      }
      await loadAll();
      success(t("common.deleteSuccess"));
    } catch (err) {
      const message = err instanceof Error ? err.message : t("errors.saveDataFailed");
      setError(message);
      showError(message);
    } finally {
      setDeletingRoleID(null);
    }
  }

  async function handleSaveUser(): Promise<void> {
    setSavingUser(true);
    setError(null);
    try {
      if (editingMembershipID) {
        await updateTenantUser(editingMembershipID, {
          email: userForm.email,
          full_name: userForm.full_name,
          username: userForm.username || undefined,
          phone_number: userForm.phone_number,
          password: userForm.password || undefined,
          status: userForm.status,
          is_active: userForm.is_active,
          role_ids: userForm.role_ids
        });
      } else {
        await createTenantUser({
          email: userForm.email,
          full_name: userForm.full_name,
          username: userForm.username || undefined,
          phone_number: userForm.phone_number,
          password: userForm.password,
          status: userForm.status,
          is_active: userForm.is_active,
          role_ids: userForm.role_ids
        });
      }
      setUserForm(defaultUserForm);
      setEditingMembershipID(null);
      await loadAll();
      success(editingMembershipID ? t("common.updateSuccess") : t("common.saveSuccess"));
    } catch (err) {
      const message = err instanceof Error ? err.message : t("errors.saveDataFailed");
      setError(message);
      showError(message);
    } finally {
      setSavingUser(false);
    }
  }

  function handleEditUser(user: TenantUserRoleEntry): void {
    setEditingMembershipID(user.membership_id);
    setUserForm({
      email: user.email,
      full_name: user.full_name,
      username: user.username ?? "",
      phone_number: user.whatsapp_number ?? "",
      password: "",
      status: user.status,
      is_active: user.is_active ?? user.status === "active",
      role_ids: user.roles.map((role) => role.id)
    });
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  async function handleDeleteUser(membershipID: string): Promise<void> {
    setUserToDelete(null);
    setDeletingMembershipID(membershipID);
    setError(null);
    try {
      await deleteTenantUser(membershipID);
      if (editingMembershipID === membershipID) {
        setEditingMembershipID(null);
        setUserForm(defaultUserForm);
      }
      await loadAll();
      success(t("common.deleteSuccess"));
    } catch (err) {
      const message = err instanceof Error ? err.message : t("errors.saveDataFailed");
      setError(message);
      showError(message);
    } finally {
      setDeletingMembershipID(null);
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("settings.title")} 
        description={t("settings.roles.subtitle")} 
      />


      {loading ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(min(100%, 450px), 1fr))", gap: "1.5rem", alignItems: "start" }}>
        <div className="card surface">
          <h3 className="form-title">{editingRoleID ? t("settings.roles.editRole") : t("settings.roles.createRole")}</h3>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              handleSaveRole().catch(() => undefined);
            }}
          >
            <label className="form-field">
              {t("settings.roles.code")}
              <input
                className="input-control"
                value={roleForm.code}
                onChange={(event) => setRoleForm((prev) => ({ ...prev, code: event.target.value.toLowerCase() }))}
                required
              />
            </label>
            <label className="form-field">
              {t("settings.roles.name")}
              <input
                className="input-control"
                value={roleForm.name}
                onChange={(event) => setRoleForm((prev) => ({ ...prev, name: event.target.value }))}
                required
              />
            </label>
            <div className="form-field">
              <span style={{ display: "block", marginBottom: "0.5rem" }}>{t("settings.roles.permissions")}</span>
              <div style={{ 
                maxHeight: "350px", 
                overflowY: "auto", 
                border: "1px solid var(--border)", 
                borderRadius: "var(--radius-sm)", 
                padding: "0.5rem",
                background: "var(--surface-sunken)"
              }}>
                <select
                  className="input-control"
                  multiple
                  style={{ border: "none", width: "100%", height: "auto", minHeight: "300px" }}
                  value={roleForm.permission_ids}
                  onChange={(event) =>
                    setRoleForm((prev) => ({
                      ...prev,
                      permission_ids: Array.from(event.target.selectedOptions).map((option) => option.value)
                    }))
                  }
                >
                  {groupedPermissions.map((group) => (
                    <optgroup key={group.moduleCode} label={formatPermissionGroup(group.moduleCode)}>
                      {group.items.map((permission) => (
                        <option key={permission.id} value={permission.id} style={{ padding: "4px 8px" }}>
                          {formatPermissionLabel(permission)}
                        </option>
                      ))}
                    </optgroup>
                  ))}
                </select>
              </div>
              <small className="page-subtitle" style={{ marginTop: "0.5rem", display: "block" }}>
                Tahan Ctrl (atau Cmd) untuk memilih beberapa hak akses.
              </small>
            </div>
            <button className="btn btn-primary" type="submit" disabled={savingRole}>
              {savingRole ? t("common.loading") : editingRoleID ? t("settings.roles.updateRole") : t("settings.roles.createRole")}
            </button>
          </form>

          <div className="data-table-wrap table-mobile-stack">
            <table className="data-table">
              <thead>
                <tr>
                  <th>{t("settings.roles.code")}</th>
                  <th>{t("settings.roles.name")}</th>
                  <th>{t("settings.roles.permissions")}</th>
                  <th>{t("settings.roles.action")}</th>
                </tr>
              </thead>
              <tbody>
                {roles.map((role) => (
                  <tr key={role.id}>
                    <td data-label={t("settings.roles.code")}>{role.code}</td>
                    <td data-label={t("settings.roles.name")}>{role.name}</td>
                    <td data-label={t("settings.roles.permissions")}>{role.permission_ids?.length ?? 0}</td>
                    <td data-label={t("settings.roles.action")}>
                      <div className="table-actions">
                        <button className="btn btn-ghost-inline" type="button" onClick={() => handleEditRole(role)} disabled={role.is_system}>
                          {t("common.edit")}
                        </button>
                        <button
                          className="btn btn-ghost-inline danger"
                          type="button"
                          onClick={() => setRoleToDelete(role)}
                          disabled={role.is_system || deletingRoleID === role.id}
                        >
                          {deletingRoleID === role.id ? t("common.loading") : t("common.delete")}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {!roles.length ? (
                  <tr>
                    <td colSpan={4}>{t("common.noItems")}</td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </div>

        <div className="card surface">
          <h3 className="form-title">{editingMembershipID ? t("settings.users.editUser") : t("settings.users.createUser")}</h3>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              handleSaveUser().catch(() => undefined);
            }}
          >
            <label className="form-field">
              {t("settings.users.fullName")}
              <input
                className="input-control"
                value={userForm.full_name}
                onChange={(event) => setUserForm((prev) => ({ ...prev, full_name: event.target.value }))}
                required
                autoComplete="name"
              />
            </label>
            <label className="form-field">
              {t("settings.users.email")}
              <input
                className="input-control"
                type="email"
                value={userForm.email}
                onChange={(event) => setUserForm((prev) => ({ ...prev, email: event.target.value }))}
                required
                autoComplete="email"
              />
            </label>
            <label className="form-field">
              {t("settings.users.username")}
              <input
                className="input-control"
                value={userForm.username}
                onChange={(event) => setUserForm((prev) => ({ ...prev, username: event.target.value.toLowerCase().replace(/\s/g, '') }))}
                placeholder="e.g. jdoe"
                autoComplete="username"
              />
            </label>
            <label className="form-field">
              {t("settings.users.phone")}
              <input
                className="input-control"
                value={userForm.phone_number}
                onChange={(event) => setUserForm((prev) => ({ ...prev, phone_number: event.target.value }))}
                placeholder="e.g. 628123456789"
                autoComplete="tel"
              />
            </label>
            <label className="form-field">
              {t("settings.users.password")}
              <input
                className="input-control"
                type="password"
                value={userForm.password}
                onChange={(event) => setUserForm((prev) => ({ ...prev, password: event.target.value }))}
                placeholder={editingMembershipID ? t("settings.users.passwordHint") : ""}
                required={!editingMembershipID}
                autoComplete="new-password"
              />
              <PasswordStrength password={userForm.password} />
            </label>
            <label className="form-field">
              {t("settings.users.status")}
              <select
                className="input-control"
                value={userForm.status}
                onChange={(event) => setUserForm((prev) => ({ ...prev, status: event.target.value }))}
              >
                <option value="active">{t("settings.users.statusActive")}</option>
                <option value="invited">{t("settings.users.statusInvited")}</option>
                <option value="suspended">{t("settings.users.statusSuspended")}</option>
              </select>
            </label>
            <label className="form-field">
              {t("settings.users.enabled")}
              <select
                className="input-control"
                value={userForm.is_active ? "1" : "0"}
                onChange={(event) => setUserForm((prev) => ({ ...prev, is_active: event.target.value === "1" }))}
              >
                <option value="1">{t("common.enabled")}</option>
                <option value="0">{t("common.disabled")}</option>
              </select>
            </label>
            <label className="form-field">
              {t("settings.roles.assignedRoles")}
              <select
                className="input-control"
                multiple
                value={userForm.role_ids}
                onChange={(event) =>
                  setUserForm((prev) => ({
                    ...prev,
                    role_ids: Array.from(event.target.selectedOptions).map((option) => option.value)
                  }))
                }
              >
                {roles.map((role) => (
                  <option key={role.id} value={role.id}>
                    {role.name} ({role.code})
                  </option>
                ))}
              </select>
            </label>
            <button className="btn btn-primary" type="submit" disabled={savingUser}>
              {savingUser ? t("common.loading") : editingMembershipID ? t("settings.users.updateUser") : t("settings.users.createUser")}
            </button>
          </form>

          <div className="data-table-wrap table-mobile-stack">
            <table className="data-table">
              <thead>
                <tr>
                  <th>{t("settings.users.fullName")}</th>
                  <th>{t("settings.users.email")}</th>
                  <th>{t("settings.users.status")}</th>
                  <th>{t("settings.roles.assignedRoles")}</th>
                  <th>{t("settings.channel.whatsapp")}</th>
                  <th>{t("settings.roles.action")}</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.membership_id}>
                    <td data-label={t("settings.users.fullName")}>{user.full_name}</td>
                    <td data-label={t("settings.users.email")}>{user.email}</td>
                    <td data-label={t("settings.users.status")}>{user.status}</td>
                    <td data-label={t("settings.roles.assignedRoles")}>{user.roles.map((role) => role.code).join(", ") || "-"}</td>
                    <td data-label={t("settings.channel.whatsapp")}>{user.whatsapp_number || "-"}</td>
                    <td data-label={t("settings.roles.action")}>
                      <div className="table-actions">
                        <button className="btn btn-ghost-inline" type="button" onClick={() => handleEditUser(user)}>
                          {t("common.edit")}
                        </button>
                        <button
                          className="btn btn-ghost-inline danger"
                          type="button"
                          onClick={() => setUserToDelete(user)}
                          disabled={deletingMembershipID === user.membership_id}
                        >
                          {deletingMembershipID === user.membership_id ? t("common.loading") : t("common.delete")}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {!users.length ? (
                  <tr>
                    <td colSpan={5}>{t("common.noItems")}</td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <DeleteConfirmModal
        isOpen={!!roleToDelete}
        title={t("common.delete")}
        message={`${t("common.delete")} role "${roleToDelete?.name}"?`}
        isLoading={deletingRoleID !== null}
        onConfirm={() => roleToDelete && handleDeleteRole(roleToDelete.id).catch(() => undefined)}
        onCancel={() => setRoleToDelete(null)}
      />

      <DeleteConfirmModal
        isOpen={!!userToDelete}
        title={t("common.delete")}
        message={`${t("common.delete")} user "${userToDelete?.full_name}"?`}
        isLoading={deletingMembershipID !== null}
        onConfirm={() => userToDelete && handleDeleteUser(userToDelete.membership_id).catch(() => undefined)}
        onCancel={() => setUserToDelete(null)}
      />

      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </section>
  );
}
