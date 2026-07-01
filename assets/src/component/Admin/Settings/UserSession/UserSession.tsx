import { Box, FormControl, FormControlLabel, Link, ListItemText, Stack, Switch, Typography } from "@mui/material";
import { useContext, useMemo } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { isTrueVal } from "../../../../session/utils.ts";
import SizeInput from "../../../Common/SizeInput.tsx";
import { DenseFilledTextField, DenseSelect } from "../../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import SettingForm, { ProChip } from "../../../Pages/Setting/SettingForm.tsx";
import { Code } from "../../../Common/Code.tsx";
import GroupSelectionInput from "../../Common/GroupSelectionInput.tsx";
import SharesInput from "../../Common/SharesInput.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../Settings.tsx";
import { SettingContext } from "../SettingWrapper.tsx";
import SSOSettings from "./SSOSettings.tsx";

const UserSession = () => {
  const { t } = useTranslation("dashboard");
  const { formRef, setSettings, values } = useContext(SettingContext);

  const defaultSymbolics = useMemo(() => {
    let result: number[] = [];
    try {
      result = JSON.parse(values?.default_symbolics ?? "[]");
    } catch (e) {
      console.error(e);
    }
    return result;
  }, [values?.default_symbolics]);

  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      <Stack spacing={5}>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.accountManagement")}
          </Typography>
          <SettingSectionContent>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.register_enabled)}
                      onChange={(e) =>
                        setSettings({
                          register_enabled: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.allowNewRegistrations")}
                />
                <NoMarginHelperText>{t("settings.allowNewRegistrationsDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.email_active)}
                      onChange={(e) =>
                        setSettings({
                          email_active: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.emailActivation")}
                />
                <NoMarginHelperText>
                  <Trans
                    i18nKey="settings.emailActivationDes"
                    ns={"dashboard"}
                    components={[<Link href={"/admin/settings?tab=email"} />]}
                  />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.authn_enabled)}
                      onChange={(e) =>
                        setSettings({
                          authn_enabled: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.webauthn")}
                />
                <NoMarginHelperText>{t("settings.webauthnDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.expose_user_email)}
                      onChange={(e) =>
                        setSettings({
                          expose_user_email: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.exposeUserEmail")}
                />
                <NoMarginHelperText>{t("settings.exposeUserEmailDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.defaultGroup")} lgWidth={5}>
              <FormControl>
                <GroupSelectionInput
                  value={values.default_group}
                  onChange={(g) =>
                    setSettings({
                      default_group: g,
                    })
                  }
                />
                <NoMarginHelperText>{t("settings.defaultGroupDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.defaultSymbolics")} lgWidth={5} pro>
              <FormControl>
                <SharesInput />
                <NoMarginHelperText>
                  <Trans
                    i18nKey="settings.defaultSymbolicsDes"
                    ns={"dashboard"}
                    components={[<Link component={RouterLink} to={"/admin/share"} />]}
                  />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("vas.filterEmailProvider")} lgWidth={5} pro>
              <FormControl>
                <DenseSelect value={0}>
                  {["filterEmailProviderDisabled", "filterEmailProviderWhitelist", "filterEmailProviderBlacklist"].map(
                    (v, i) => (
                      <SquareMenuItem value={i.toString()}>
                        <ListItemText
                          slotProps={{
                            primary: { variant: "body2" },
                          }}
                        >
                          {t(`vas.${v}`)}
                        </ListItemText>
                      </SquareMenuItem>
                    ),
                  )}
                </DenseSelect>
                <NoMarginHelperText>{t("vas.filterEmailProviderDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm lgWidth={5} pro>
              <FormControl fullWidth>
                <FormControlLabel
                  control={<Switch checked={false} />}
                  label={
                    <>
                      {t("vas.disableSubAddressEmail")}
                      <ProChip label="Pro" color="primary" size="small" />
                    </>
                  }
                />
                <NoMarginHelperText>
                  <Trans i18nKey="vas.disableSubAddressEmailDes" ns={"dashboard"} components={[<Code />]} />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.smsLogin")}
          </Typography>
          <SettingSectionContent>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.sms_login_enabled)}
                      onChange={(e) =>
                        setSettings({
                          sms_login_enabled: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.smsLoginEnabled")}
                />
                <NoMarginHelperText>{t("settings.smsLoginEnabledDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.smsSecretName")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value={values.sms_secret_name ?? ""}
                  onChange={(e) =>
                    setSettings({
                      sms_secret_name: e.target.value,
                    })
                  }
                />
                <NoMarginHelperText>{t("settings.smsSecretNameDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.smsSecretKey")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value={values.sms_secret_key ?? ""}
                  type={"password"}
                  autoComplete={"new-password"}
                  onChange={(e) =>
                    setSettings({
                      sms_secret_key: e.target.value,
                    })
                  }
                />
                <NoMarginHelperText>{t("settings.smsSecretKeyDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.smsSignName")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value={values.sms_sign_name ?? ""}
                  onChange={(e) =>
                    setSettings({
                      sms_sign_name: e.target.value,
                    })
                  }
                />
                <NoMarginHelperText>{t("settings.smsSignNameDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.smsEndpoint")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value={values.sms_endpoint ?? ""}
                  onChange={(e) =>
                    setSettings({
                      sms_endpoint: e.target.value,
                    })
                  }
                />
                <NoMarginHelperText>{t("settings.smsEndpointDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.smsTemplate")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value={values.sms_template ?? ""}
                  multiline
                  minRows={2}
                  onChange={(e) =>
                    setSettings({
                      sms_template: e.target.value,
                    })
                  }
                />
                <NoMarginHelperText>
                  <Trans i18nKey="settings.smsTemplateDes" ns={"dashboard"} components={[<Code />]} />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.smsCodeTtl")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value={values.sms_code_ttl ?? ""}
                  type={"number"}
                  inputProps={{ step: 1, min: 60 }}
                  onChange={(e) =>
                    setSettings({
                      sms_code_ttl: e.target.value,
                    })
                  }
                />
                <NoMarginHelperText>{t("settings.smsCodeTtlDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.smsSendInterval")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value={values.sms_send_interval ?? ""}
                  type={"number"}
                  inputProps={{ step: 1, min: 30 }}
                  onChange={(e) =>
                    setSettings({
                      sms_send_interval: e.target.value,
                    })
                  }
                />
                <NoMarginHelperText>{t("settings.smsSendIntervalDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.sms_auto_register)}
                      onChange={(e) =>
                        setSettings({
                          sms_auto_register: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.smsAutoRegister")}
                />
                <NoMarginHelperText>{t("settings.smsAutoRegisterDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
        <SettingSection>
          <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
            {t("settings.thirdPartySignIn")} <ProChip label="Pro" color="primary" size="small" />
          </Typography>
          <SettingSectionContent>
            <SettingForm lgWidth={5}>
              <SSOSettings />
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.avatar")}
          </Typography>
          <SettingSectionContent>
            <SettingForm title={t("settings.avatarFilePath")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value={values.avatar_path}
                  onChange={(e) =>
                    setSettings({
                      avatar_path: e.target.value,
                    })
                  }
                  required
                />
                <NoMarginHelperText>{t("settings.avatarFilePathDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.avatarSize")} lgWidth={5}>
              <FormControl>
                <SizeInput
                  variant={"outlined"}
                  required
                  label={t("application:navbar.minimum")}
                  value={parseInt(values.avatar_size) ?? 0}
                  onChange={(e) =>
                    setSettings({
                      avatar_size: e.toString(),
                    })
                  }
                />
                <NoMarginHelperText>{t("settings.avatarSizeDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.avatarImageSize")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value={values.avatar_size_l}
                  onChange={(e) =>
                    setSettings({
                      avatar_size_l: e.target.value,
                    })
                  }
                  type={"number"}
                  inputProps={{ step: 1, min: 1 }}
                  required
                />
                <NoMarginHelperText>{t("settings.avatarImageSizeDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.gravatarServer")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value={values.gravatar_server}
                  onChange={(e) =>
                    setSettings({
                      gravatar_server: e.target.value,
                    })
                  }
                  required
                />
                <NoMarginHelperText>{t("settings.gravatarServerDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
      </Stack>
    </Box>
  );
};

export default UserSession;
