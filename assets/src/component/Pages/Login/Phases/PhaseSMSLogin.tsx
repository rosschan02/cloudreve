import { Box, Button, FormControl, Link, Stack } from "@mui/material";
import { enqueueSnackbar } from "notistack";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendSMSCode } from "../../../../api/api.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";
import { OutlineIconTextField } from "../../../Common/Form/OutlineIconTextField.tsx";
import Numbers from "../../../Icons/Numbers.tsx";
import PhoneLaptopOutlined from "../../../Icons/PhoneLaptopOutlined.tsx";
import { Control } from "../Signin/SignIn.tsx";

interface PhaseSMSLoginProps {
  phone: string;
  setPhone: (phone: string) => void;
  code: string;
  setCode: (code: string) => void;
  control?: Control;
  onBack: () => void;
}

const RESEND_SECONDS = 60;
const phonePattern = /^1[3-9]\d{9}$/;

const PhaseSMSLogin = ({ phone, setPhone, code, setCode, control, onBack }: PhaseSMSLoginProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const [countdown, setCountdown] = useState(0);
  const [sending, setSending] = useState(false);
  const timer = useRef<ReturnType<typeof setInterval>>();

  useEffect(() => {
    return () => {
      if (timer.current) clearInterval(timer.current);
    };
  }, []);

  const startCountdown = useCallback(() => {
    setCountdown(RESEND_SECONDS);
    timer.current = setInterval(() => {
      setCountdown((c) => {
        if (c <= 1 && timer.current) {
          clearInterval(timer.current);
          return 0;
        }
        return c - 1;
      });
    }, 1000);
  }, []);

  const onSendCode = useCallback(async () => {
    if (!phonePattern.test(phone)) {
      enqueueSnackbar({
        message: t("login.invalidPhone"),
        variant: "warning",
        action: DefaultCloseAction,
      });
      return;
    }
    try {
      setSending(true);
      await dispatch(sendSMSCode({ phone }));
      startCountdown();
      enqueueSnackbar({
        message: t("login.smsCodeSent"),
        variant: "success",
        action: DefaultCloseAction,
      });
    } catch (e) {
      // Snackbar handled by request layer.
    } finally {
      setSending(false);
    }
  }, [dispatch, phone, startCountdown, t]);

  return (
    <>
      <FormControl variant="standard" margin="normal" required fullWidth>
        <OutlineIconTextField
          label={t("login.phoneNumber")}
          variant={"outlined"}
          inputProps={{
            id: "phone",
            type: "tel",
            name: "phone",
            required: "true",
            maxLength: 11,
          }}
          onChange={(e) => setPhone(e.target.value.trim())}
          icon={<PhoneLaptopOutlined />}
          autoComplete={"tel"}
          value={phone}
          autoFocus
        />
      </FormControl>
      <FormControl variant="standard" margin="normal" required fullWidth>
        <Stack direction={"row"} spacing={1} alignItems={"stretch"}>
          <OutlineIconTextField
            sx={{ flexGrow: 1 }}
            label={t("login.smsCode")}
            variant={"outlined"}
            inputProps={{
              id: "sms_code",
              name: "sms_code",
              required: "true",
              inputMode: "numeric",
              maxLength: 6,
            }}
            onChange={(e) => setCode(e.target.value.trim())}
            icon={<Numbers />}
            value={code}
          />
          <Button
            variant={"outlined"}
            sx={{ flexShrink: 0, minWidth: 120 }}
            disabled={countdown > 0 || sending}
            onClick={onSendCode}
          >
            {countdown > 0 ? t("login.resendAfter", { sec: countdown }) : t("login.getSmsCode")}
          </Button>
        </Stack>
      </FormControl>
      {control?.submit}
      <Box sx={{ mt: 2, textAlign: "center" }}>
        <Link component={"button"} type={"button"} underline="hover" variant={"body2"} onClick={onBack}>
          {t("login.backToEmailLogin")}
        </Link>
      </Box>
    </>
  );
};

export default PhaseSMSLogin;
