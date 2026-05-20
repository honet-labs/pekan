import { useAccessStore } from "../../core/access/access-store";
import { useI18n } from "../../core/i18n/i18n";

type Props = {
  children: JSX.Element;
  feature: string;
};

export function FeatureGuard({ children, feature }: Props): JSX.Element {
  const access = useAccessStore();
  const { t } = useI18n();
  if (!access.features.has(feature)) {
    return <p>{t("common.featureDisabled")}</p>;
  }
  return children;
}

