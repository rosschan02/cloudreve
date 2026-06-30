import {
  Box,
  Button,
  Chip,
  CircularProgress,
  Collapse,
  DialogContent,
  Divider,
  List,
  ListItem,
  ListItemText,
  Stack,
  TextField,
  Typography,
} from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  applyGroupShare,
  GroupShareApplicant,
  GroupShareEntry,
  listGroupShareApplications,
  listGroupShares,
  reviewGroupShare,
} from "../../../api/api.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";

export interface GroupShareManageProps {
  open: boolean;
  onClose: () => void;
}

const GroupShareManage = ({ open, onClose }: GroupShareManageProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(false);
  const [entries, setEntries] = useState<GroupShareEntry[]>([]);
  const [busyId, setBusyId] = useState<string>("");
  const [expanded, setExpanded] = useState<string>("");
  const [applications, setApplications] = useState<Record<string, GroupShareApplicant[]>>({});

  // Apply form state
  const [applyTarget, setApplyTarget] = useState<GroupShareEntry | null>(null);
  const [realName, setRealName] = useState("");
  const [reason, setReason] = useState("");
  const [applying, setApplying] = useState(false);

  const refresh = useCallback(() => {
    setLoading(true);
    dispatch(listGroupShares())
      .then((res) => setEntries(res?.groups ?? []))
      .finally(() => setLoading(false));
  }, [dispatch]);

  useEffect(() => {
    if (open) {
      refresh();
      setExpanded("");
      setApplications({});
    }
  }, [open, refresh]);

  const openApply = useCallback((entry: GroupShareEntry) => {
    setApplyTarget(entry);
    setRealName("");
    setReason("");
  }, []);

  const submitApply = useCallback(() => {
    if (!applyTarget || !realName.trim() || !reason.trim()) {
      return;
    }
    setApplying(true);
    dispatch(applyGroupShare(applyTarget.id, realName.trim(), reason.trim()))
      .then(() => {
        setApplyTarget(null);
        refresh();
      })
      .finally(() => setApplying(false));
  }, [dispatch, applyTarget, realName, reason, refresh]);

  const loadApplications = useCallback(
    (id: string) => {
      if (expanded === id) {
        setExpanded("");
        return;
      }
      setExpanded(id);
      dispatch(listGroupShareApplications(id)).then((res) => {
        setApplications((p) => ({ ...p, [id]: res ?? [] }));
      });
    },
    [dispatch, expanded],
  );

  const onReview = useCallback(
    (id: string, userId: string, approve: boolean) => {
      setBusyId(id + userId);
      dispatch(reviewGroupShare(id, userId, approve))
        .then(() => {
          dispatch(listGroupShareApplications(id)).then((res) => {
            setApplications((p) => ({ ...p, [id]: res ?? [] }));
          });
          refresh();
        })
        .finally(() => setBusyId(""));
    },
    [dispatch, refresh],
  );

  const joinable = entries.filter((e) => e.status === "joinable" || e.status === "pending");
  const approverGroups = entries.filter((e) => e.is_approver);

  return (
    <>
      <DraggableDialog
        title={t("application:navbar.groupShareManage")}
        dialogProps={{ open, onClose: () => onClose(), fullWidth: true, maxWidth: "sm" }}
        showActions
        hideOk
        showCancel
        cancelText={t("common:close")}
      >
        <DialogContent>
          {loading && entries.length === 0 ? (
            <Box sx={{ display: "flex", justifyContent: "center", py: 4 }}>
              <CircularProgress />
            </Box>
          ) : (
            <Stack spacing={3}>
              <Box>
                <Typography variant="subtitle2" gutterBottom>
                  {t("application:navbar.groupShareJoinable")}
                </Typography>
                {joinable.length === 0 ? (
                  <Typography variant="body2" color="text.secondary">
                    {t("application:navbar.groupShareNoneJoinable")}
                  </Typography>
                ) : (
                  <List dense>
                    {joinable.map((e) => (
                      <ListItem
                        key={e.id}
                        secondaryAction={
                          e.status === "pending" ? (
                            <Chip size="small" label={t("application:navbar.groupSharePending")} />
                          ) : (
                            <Button size="small" variant="outlined" onClick={() => openApply(e)}>
                              {t("application:navbar.groupShareApply")}
                            </Button>
                          )
                        }
                      >
                        <ListItemText primary={e.name} />
                      </ListItem>
                    ))}
                  </List>
                )}
              </Box>

              {approverGroups.length > 0 && (
                <Box>
                  <Divider sx={{ mb: 2 }} />
                  <Typography variant="subtitle2" gutterBottom>
                    {t("application:navbar.groupShareReview")}
                  </Typography>
                  <List dense>
                    {approverGroups.map((e) => (
                      <Box key={e.id}>
                        <ListItem
                          secondaryAction={
                            <Button size="small" onClick={() => loadApplications(e.id)}>
                              {e.pending_count > 0
                                ? t("application:navbar.groupSharePendingCount", { count: e.pending_count })
                                : t("application:navbar.groupShareNoPending")}
                            </Button>
                          }
                        >
                          <ListItemText primary={e.name} />
                        </ListItem>
                        <Collapse in={expanded === e.id} unmountOnExit>
                          <List dense sx={{ pl: 2 }}>
                            {(applications[e.id] ?? []).length === 0 ? (
                              <Typography variant="body2" color="text.secondary" sx={{ pl: 2, py: 1 }}>
                                {t("application:navbar.groupShareNoPending")}
                              </Typography>
                            ) : (
                              (applications[e.id] ?? []).map((a) => (
                                <ListItem
                                  key={a.user.id}
                                  alignItems="flex-start"
                                  secondaryAction={
                                    <Stack direction="row" spacing={1}>
                                      <Button
                                        size="small"
                                        variant="contained"
                                        disabled={busyId === e.id + a.user.id}
                                        onClick={() => onReview(e.id, a.user.id, true)}
                                      >
                                        {t("application:navbar.groupShareApprove")}
                                      </Button>
                                      <Button
                                        size="small"
                                        color="error"
                                        disabled={busyId === e.id + a.user.id}
                                        onClick={() => onReview(e.id, a.user.id, false)}
                                      >
                                        {t("application:navbar.groupShareReject")}
                                      </Button>
                                    </Stack>
                                  }
                                >
                                  <ListItemText
                                    primary={
                                      <span>
                                        {a.real_name || a.user.nickname}
                                        <Typography component="span" variant="body2" color="text.secondary">
                                          {"  ("}
                                          {a.user.nickname}
                                          {a.user.email ? ` / ${a.user.email}` : ""}
                                          {")"}
                                        </Typography>
                                      </span>
                                    }
                                    secondary={
                                      a.reason
                                        ? `${t("application:navbar.groupShareReason")}: ${a.reason}`
                                        : undefined
                                    }
                                  />
                                </ListItem>
                              ))
                            )}
                          </List>
                        </Collapse>
                      </Box>
                    ))}
                  </List>
                </Box>
              )}
            </Stack>
          )}
        </DialogContent>
      </DraggableDialog>

      {/* Apply form: real name + reason */}
      <DraggableDialog
        title={t("application:navbar.groupShareApplyTitle", { name: applyTarget?.name ?? "" })}
        dialogProps={{ open: !!applyTarget, onClose: () => setApplyTarget(null), fullWidth: true, maxWidth: "xs" }}
        showActions
        okText={t("application:navbar.groupShareApply")}
        onAccept={submitApply}
        loading={applying}
        disabled={!realName.trim() || !reason.trim()}
        showCancel
      >
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            <TextField
              label={t("application:navbar.groupShareRealName")}
              value={realName}
              onChange={(e) => setRealName(e.target.value)}
              required
              fullWidth
              inputProps={{ maxLength: 255 }}
            />
            <TextField
              label={t("application:navbar.groupShareReason")}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              required
              fullWidth
              multiline
              minRows={3}
              inputProps={{ maxLength: 1000 }}
            />
          </Stack>
        </DialogContent>
      </DraggableDialog>
    </>
  );
};

export default GroupShareManage;
