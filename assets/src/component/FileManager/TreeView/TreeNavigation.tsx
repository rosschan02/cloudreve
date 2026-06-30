import { Box, Collapse, Fade } from "@mui/material";

import { ChevronRight, ExpandMore } from "@mui/icons-material";
import { TreeView } from "@mui/x-tree-view";
import React, { useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { TransitionGroup } from "react-transition-group";
import { defaultPath, defaultSharedWithMePath, defaultTrashPath } from "../../../hooks/useNavigation.tsx";
import { GroupShareEntry, listGroupShares } from "../../../api/api.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import SessionManager, { UserSettings } from "../../../session";
import CrUri, { Filesystem } from "../../../util/uri.ts";
import People from "../../Icons/People.tsx";
import SideNavItem from "../../Frame/NavBar/SideNavItem.tsx";
import { FileManagerIndex } from "../FileManager.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";
import GroupShareManage from "../Dialogs/GroupShareManage.tsx";
import Pinned, { usePinned } from "./Pinned.tsx";
import TreeFiles from "./TreeFiles.tsx";

export interface TreeNavigationProps {
  scrollRef?: React.MutableRefObject<HTMLElement | undefined>;
  index?: number;
  hideWithDrawer?: boolean;
  disableSharedWithMe?: boolean;
  disableTrash?: boolean;
}

const TreeNavigation = React.memo(
  ({ index = 0, scrollRef, hideWithDrawer, disableSharedWithMe, disableTrash }: TreeNavigationProps) => {
    const base = useAppSelector((s) => s.fileManager[index].path_root);
    const path = useAppSelector((s) => s.fileManager[index].pure_path_with_category);
    const currentFs = useAppSelector((s) => s.fileManager[index].current_fs);
    const elements = useAppSelector((s) => s.fileManager[index].path_elements);
    const drawerOpen = useAppSelector((s) => s.globalState.drawerOpen);
    const [expanded, setExpanded] = React.useState<string[]>([]);

    useEffect(() => {
      const res: string[] = [];
      if (path) {
        const p = new CrUri(path);
        if (p.is_search()) {
          return;
        }
      }
      if (base && elements) {
        const b = new CrUri(base);
        res.push(base);
        elements.forEach((element) => {
          b.join(element);
          res.push(b.toString());
        });
      }
      if (SessionManager.getWithFallback(UserSettings.TreeViewAutoExpand)) {
        setExpanded((e) => [...new Set([...e, ...res])]);
      }
    }, [path, base, elements]);

    const pinned = usePinned();
    const alreadyPinned = base && pinned && pinned.find((p) => p.uri == base);
    const showShareTree = base && currentFs && currentFs == Filesystem.share && !alreadyPinned;

    useEffect(() => {
      if (showShareTree && scrollRef && scrollRef.current) {
        scrollRef.current?.scrollTo({ top: 0, behavior: "smooth" });
      }
    }, [showShareTree]);

    const loginUser = SessionManager.currentLoginOrNull();
    const isLogin = !!loginUser;

    // Group share areas the current user can access or apply to (decoupled from primary group).
    const dispatch = useAppDispatch();
    const { t } = useTranslation("application");
    const [groupShares, setGroupShares] = React.useState<GroupShareEntry[]>([]);
    const [manageOpen, setManageOpen] = React.useState(false);
    const isMain = index == FileManagerIndex.main;

    const refreshGroupShares = useCallback(() => {
      if (!isLogin || !isMain) {
        return;
      }
      dispatch(listGroupShares())
        .then((res) => setGroupShares(res?.groups ?? []))
        .catch(() => {});
    }, [dispatch, isLogin, isMain]);

    useEffect(() => {
      refreshGroupShares();
    }, [refreshGroupShares]);

    const joinedGroupShares = groupShares.filter((g) => !!g.uri);

    return (
      <FmIndexContext.Provider value={index}>
        <Box sx={{ width: "100%" }}>
          <Fade in={drawerOpen || !hideWithDrawer} unmountOnExit>
            <TreeView
              selected={path ?? ""}
              defaultCollapseIcon={<ExpandMore />}
              defaultExpandIcon={<ChevronRight />}
              expanded={expanded}
              onNodeToggle={(_event, nodeIds: string[]) => {
                setExpanded(nodeIds);
              }}
            >
              <TransitionGroup>
                {showShareTree && (
                  <Collapse key={"share"}>
                    <TreeFiles level={0} path={base} elements={elements} />
                  </Collapse>
                )}
                {isLogin && (
                  <>
                    <TreeFiles
                      canDrop
                      level={0}
                      path={defaultPath}
                      key={defaultPath}
                      elements={currentFs == Filesystem.my ? elements : undefined}
                    />
                    {index == FileManagerIndex.main && (
                      <>
                        <TreeFiles
                          flatten
                          level={0}
                          path={"cloudreve://my/?category=image"}
                          key={"cloudreve://my/?category=image"}
                        />
                        <TreeFiles
                          flatten
                          level={0}
                          path={"cloudreve://my/?category=video"}
                          key={"cloudreve://my/?category=video"}
                        />
                        <TreeFiles
                          flatten
                          level={0}
                          path={"cloudreve://my/?category=audio"}
                          key={"cloudreve://my/?category=audio"}
                        />
                        <TreeFiles
                          flatten
                          level={0}
                          path={"cloudreve://my/?category=document"}
                          key={"cloudreve://my/?category=document"}
                        />
                      </>
                    )}
                    {!disableSharedWithMe && (
                      <TreeFiles level={0} path={defaultSharedWithMePath} key={defaultSharedWithMePath} />
                    )}
                    {isMain &&
                      joinedGroupShares.map((g) => (
                        <TreeFiles
                          level={0}
                          canDrop
                          path={g.uri as string}
                          key={g.uri}
                          labelOverwrite={g.name}
                          elements={currentFs == Filesystem.group && g.uri == base ? elements : undefined}
                        />
                      ))}
                    {isMain && groupShares.length > 0 && (
                      <SideNavItem
                        level={0}
                        icon={<People fontSize="small" color="action" />}
                        label={t("navbar.groupShareManage")}
                        onClick={() => setManageOpen(true)}
                      />
                    )}
                    {!disableTrash && (
                      <TreeFiles
                        level={0}
                        flatten
                        canDrop
                        key={defaultTrashPath}
                        path={defaultTrashPath}
                        elements={currentFs == Filesystem.trash ? elements : undefined}
                      />
                    )}
                    <Pinned />
                  </>
                )}
              </TransitionGroup>
            </TreeView>
          </Fade>
        </Box>
        {isMain && (
          <GroupShareManage
            open={manageOpen}
            onClose={() => {
              setManageOpen(false);
              refreshGroupShares();
            }}
          />
        )}
      </FmIndexContext.Provider>
    );
  },
);

export default TreeNavigation;
