import { useAccessStore } from "../../core/access/access-store";
import { useI18n } from "../../core/i18n/i18n";

type Props = {
  children: JSX.Element;
  permission: string;
};

export function PermissionGuard({ children, permission }: Props): JSX.Element {
  const access = useAccessStore();
  const { t } = useI18n();
  if (!access.permissions.has(permission)) {
    return <p>{t("common.permissionDenied")}</p>;
  }
  return children;
}

