import { Box } from "@mui/material";
import * as React from "react";
import { createContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { getGroupDetail, upsertGroup } from "../../../../api/api.ts";
import { GroupEnt, StoragePolicy } from "../../../../api/dashboard.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import FacebookCircularProgress from "../../../Common/CircularProgress.tsx";
import { SavingFloat } from "../../Settings/SettingWrapper.tsx";

export interface GroupSettingWrapperProps {
  groupID: number;
  children: React.ReactNode;
  onGroupChange: (group: GroupEnt) => void;
}

export interface GroupSettingContextProps {
  values: GroupEnt;
  setGroup: (f: (p: GroupEnt) => GroupEnt) => void;
  formRef?: React.RefObject<HTMLFormElement>;
  // Hashid of a newly picked group share root owner (sent to the server to be decoded).
  shareRootOwnerHash?: string;
  setShareRootOwnerHash?: (v?: string) => void;
}

const defaultGroup: GroupEnt = {
  id: 0,
  name: "",
  edges: {},
};

export const GroupSettingContext = createContext<GroupSettingContextProps>({
  values: { ...defaultGroup },
  setGroup: () => {},
});

const groupValueFilter = (group: GroupEnt): GroupEnt => {
  return {
    ...group,
    edges: {
      storage_policies: {
        id: group.edges.storage_policies?.id ?? 0,
      } as StoragePolicy,
    },
  };
};

const GroupSettingWrapper = ({ groupID, children, onGroupChange }: GroupSettingWrapperProps) => {
  const dispatch = useAppDispatch();
  const { t } = useTranslation("dashboard");
  const [values, setValues] = useState<GroupEnt>({
    ...defaultGroup,
  });
  const [modifiedValues, setModifiedValues] = useState<GroupEnt>({
    ...defaultGroup,
  });
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [shareRootOwnerHash, setShareRootOwnerHash] = useState<string | undefined>(undefined);
  const formRef = useRef<HTMLFormElement>(null);

  const showSaveButton = useMemo(() => {
    return JSON.stringify(modifiedValues) !== JSON.stringify(values) || shareRootOwnerHash !== undefined;
  }, [modifiedValues, values, shareRootOwnerHash]);

  useEffect(() => {
    setLoading(true);
    dispatch(getGroupDetail(groupID))
      .then((res) => {
        setValues(groupValueFilter(res));
        setModifiedValues(groupValueFilter(res));
        onGroupChange(groupValueFilter(res));
      })
      .finally(() => {
        setLoading(false);
      });
  }, [groupID]);

  const revert = () => {
    setModifiedValues(values);
    setShareRootOwnerHash(undefined);
  };

  const submit = () => {
    if (formRef.current) {
      if (!formRef.current.checkValidity()) {
        formRef.current.reportValidity();
        return;
      }
    }

    setSubmitting(true);
    dispatch(
      upsertGroup({
        group: { ...modifiedValues },
        share_root_owner_hash: shareRootOwnerHash,
      }),
    )
      .then((res) => {
        setValues(groupValueFilter(res));
        setModifiedValues(groupValueFilter(res));
        onGroupChange(groupValueFilter(res));
        setShareRootOwnerHash(undefined);
      })
      .finally(() => {
        setSubmitting(false);
      });
  };

  return (
    <GroupSettingContext.Provider
      value={{
        values: modifiedValues,
        setGroup: setModifiedValues,
        formRef,
        shareRootOwnerHash,
        setShareRootOwnerHash,
      }}
    >
      <SwitchTransition>
        <CSSTransition
          addEndListener={(node, done) => node.addEventListener("transitionend", done, false)}
          classNames="fade"
          key={`${loading}`}
        >
          <Box sx={{ mt: 3 }}>
            {loading && (
              <Box
                sx={{
                  pt: 20,
                  height: "100%",
                  display: "flex",
                  justifyContent: "center",
                  alignItems: "center",
                }}
              >
                <FacebookCircularProgress />
              </Box>
            )}
            {!loading && (
              <Box>
                {children}
                <SavingFloat in={showSaveButton} submitting={submitting} revert={revert} submit={submit} />
              </Box>
            )}
          </Box>
        </CSSTransition>
      </SwitchTransition>
    </GroupSettingContext.Provider>
  );
};

export default GroupSettingWrapper;
