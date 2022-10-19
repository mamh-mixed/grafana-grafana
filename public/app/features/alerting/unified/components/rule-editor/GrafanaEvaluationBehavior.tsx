import { css, cx } from '@emotion/css';
import classNames from 'classnames';
import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { RegisterOptions, useFormContext } from 'react-hook-form';

import { durationToMilliseconds, GrafanaTheme2, parseDuration, SelectableValue } from '@grafana/data';
import { secondsToHms } from '@grafana/data/src/datetime/rangeutil';
import {
  Button,
  Card,
  Field,
  Icon,
  InlineLabel,
  Input,
  InputControl,
  Label,
  Stack,
  Tooltip,
  useStyles2,
} from '@grafana/ui';
import { FolderPickerFilter } from 'app/core/components/Select/FolderPicker';
import { contextSrv } from 'app/core/core';
import { DashboardSearchHit } from 'app/features/search/types';
import { AccessControlAction, useDispatch } from 'app/types';

import { RuleNamespace } from '../../../../../types/unified-alerting';
import { useUnifiedAlertingSelector } from '../../hooks/useUnifiedAlertingSelector';
import { fetchAllPromRulesAction } from '../../state/actions';
import { RuleForm, RuleFormValues } from '../../types/rule-form';
import { checkEvaluationIntervalGlobalLimit } from '../../utils/config';
import { GRAFANA_RULES_SOURCE_NAME } from '../../utils/datasource';
import { AsyncRequestState } from '../../utils/redux';
import {
  durationValidationPattern,
  parseDurationToMilliseconds,
  positiveDurationValidationPattern,
} from '../../utils/time';
import { CollapseToggle } from '../CollapseToggle';
import { EvaluationIntervalLimitExceeded } from '../InvalidIntervalWarning';

import { GrafanaAlertStatePicker } from './GrafanaAlertStatePicker';
import { RuleEditorSection } from './RuleEditorSection';
import { containsSlashes, Folder, RuleFolderPicker } from './RuleFolderPicker';
import { SelectWithAdd } from './SelectWIthAdd';
import { checkForPathSeparator } from './util';

const MIN_TIME_RANGE_STEP_S = 10; // 10 seconds

const getIntervalForGroup = (promRules: RuleNamespace[], group: string, folder: string) => {
  const folderObj = promRules.find((ruleNameSpace) => ruleNameSpace.name === folder);
  const groupObj = folderObj?.groups.find((ruleGroup) => ruleGroup.name === group);
  const interval = groupObj?.interval ?? MINUTE;
  return interval;
};

const useGetGroups = (groupfoldersForGrafana: AsyncRequestState<RuleNamespace[]>, folderName: string) => {
  const groupOptions = useMemo(() => {
    const groupsForFolderResult: [string, RuleNamespace] | undefined = Object.entries(
      groupfoldersForGrafana?.result ?? {}
    ).find((entry) => entry[1].name === folderName);
    if (!groupfoldersForGrafana?.loading && !groupfoldersForGrafana?.error && groupfoldersForGrafana?.result) {
      return groupsForFolderResult ? groupsForFolderResult[1].groups.map((group) => group.name) : [];
    } else {
      return [];
    }
  }, [groupfoldersForGrafana, folderName]);
  return groupOptions;
};

function mapGroupsToOptions(groups: string[]): Array<SelectableValue<string>> {
  return groups.map((group) => ({ label: group, value: group }));
}

const forValidationOptions = (evaluateEvery: string): RegisterOptions => ({
  required: {
    value: true,
    message: 'Required.',
  },
  pattern: durationValidationPattern,
  validate: (value) => {
    const millisFor = parseDurationToMilliseconds(value);
    const millisEvery = parseDurationToMilliseconds(evaluateEvery);

    // 0 is a special value meaning for equals evaluation interval
    if (millisFor === 0) {
      return true;
    }

    return millisFor >= millisEvery ? true : 'For must be greater than or equal to evaluate every.';
  },
});

const useRuleFolderFilter = (existingRuleForm: RuleForm | null) => {
  const isSearchHitAvailable = useCallback(
    (hit: DashboardSearchHit) => {
      const rbacDisabledFallback = contextSrv.hasEditPermissionInFolders;

      const canCreateRuleInFolder = contextSrv.hasAccessInMetadata(
        AccessControlAction.AlertingRuleCreate,
        hit,
        rbacDisabledFallback
      );

      const canUpdateInCurrentFolder =
        existingRuleForm &&
        hit.folderId === existingRuleForm.id &&
        contextSrv.hasAccessInMetadata(AccessControlAction.AlertingRuleUpdate, hit, rbacDisabledFallback);
      return canCreateRuleInFolder || canUpdateInCurrentFolder;
    },
    [existingRuleForm]
  );

  return useCallback<FolderPickerFilter>(
    (folderHits) =>
      folderHits
        .filter(isSearchHitAvailable)
        .filter((value: DashboardSearchHit) => !containsSlashes(value.title ?? '')),
    [isSearchHitAvailable]
  );
};

function InfoIcon({ text }: { text: string }) {
  return (
    <Tooltip placement="top" content={<div>{text}</div>}>
      <Icon name="info-circle" size="xs" />
    </Tooltip>
  );
}

interface FolderAndGroupProps {
  initialFolder: RuleForm | null;
}

const useGetGroupOptionsFromFolder = (folderTilte: string) => {
  const promRules = useUnifiedAlertingSelector((state) => state.promRules);
  const groupfoldersForGrafana = promRules[GRAFANA_RULES_SOURCE_NAME];

  const groupOptions: Array<SelectableValue<string>> = mapGroupsToOptions(
    useGetGroups(groupfoldersForGrafana, folderTilte)
  );
  const groupsForFolder = groupfoldersForGrafana?.result ?? [];
  return { groupOptions, groupsForFolder };
};

const MINUTE = 60;

function FolderAndGroup({ initialFolder }: FolderAndGroupProps) {
  const {
    formState: { errors },
    watch,
    control,
  } = useFormContext<RuleFormValues>();

  const styles = useStyles2(getStyles);
  const folderFilter = useRuleFolderFilter(initialFolder);
  const dispatch = useDispatch();

  const folder = watch('folder');
  const group = watch('group');
  const [selectedGroup, setSelectedGroup] = useState(group);
  const initialRender = useRef(true);

  const { groupOptions, groupsForFolder } = useGetGroupOptionsFromFolder(folder?.title ?? '');

  useEffect(() => {
    dispatch(fetchAllPromRulesAction());
  }, [dispatch]);

  const resetGroup = useCallback(() => {
    if (group && !initialRender.current && folder?.title) {
      setSelectedGroup('');
    }
    initialRender.current = false;
  }, [initialRender, folder, setSelectedGroup, group]);

  const groupIsInGroupOptions = useCallback(
    (group_: string) => {
      return groupOptions.includes((groupInList: SelectableValue<string>) => groupInList.label === group_);
    },
    [groupOptions]
  );
  const folderTilte = folder?.title ?? '';

  useEffect(() => {
    if (folderTilte && groupOptions && !groupIsInGroupOptions(selectedGroup)) {
      resetGroup();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [folderTilte]);

  return (
    <div className={classNames([styles.flexRow, styles.alignBaseline])}>
      <Field
        label={
          <Label htmlFor="folder" description={'Select a folder to store your rule.'}>
            <Stack gap={0.5}>
              Folder
              <InfoIcon
                text={
                  'Each folder has unique folder permission. When you store multiple rules in a folder, the folder access permissions get assigned to the rules.'
                }
              />
            </Stack>
          </Label>
        }
        className={styles.formInput}
        error={errors.folder?.message}
        invalid={!!errors.folder?.message}
        data-testid="folder-picker"
      >
        <InputControl
          render={({ field: { ref, ...field } }) => (
            <RuleFolderPicker
              inputId="folder"
              {...field}
              enableCreateNew={contextSrv.hasPermission(AccessControlAction.FoldersCreate)}
              enableReset={true}
              filter={folderFilter}
              dissalowSlashes={true}
            />
          )}
          name="folder"
          rules={{
            required: { value: true, message: 'Please select a folder' },
            validate: {
              pathSeparator: (folder: Folder) => checkForPathSeparator(folder.title),
            },
          }}
        />
      </Field>

      <Field
        label="Group"
        data-testid="group-picker"
        description="Rules within the same group are evaluated after the same time interval."
        className={styles.formInput}
        error={errors.group?.message}
        invalid={!!errors.group?.message}
      >
        <InputControl
          render={({ field: { ref, ...field } }) => (
            <SelectWithAdd
              {...field}
              options={groupOptions}
              getOptionLabel={(option: SelectableValue<string>) =>
                `${option.label} -- ${secondsToHms(
                  getIntervalForGroup(groupsForFolder ?? [], option.label ?? '', folder?.title ?? '')
                )}`
              }
              width={42}
              value={selectedGroup}
              onChange={(value: string) => {
                field.onChange(value);
                setSelectedGroup(value);
              }}
            />
          )}
          name="group"
          control={control}
          rules={{
            required: { value: true, message: 'Must enter a group name' },
          }}
        />
      </Field>
    </div>
  );
}

function EvaluationIntervalInput({ initialFolder }: { initialFolder: RuleForm | null }) {
  const styles = useStyles2(getStyles);
  const [editInterval, setEditInterval] = useState(false);
  const {
    setFocus,
    register,
    formState: { errors },
    watch,
    setValue,
  } = useFormContext<RuleFormValues>();

  const group = watch('group');
  const folder = watch('folder');
  const evaluateEveryId = 'eval-every-input';
  const evaluateEvery = watch('evaluateEvery');
  const alertName = watch('name');
  const promRules = useUnifiedAlertingSelector((state) => state.promRules);
  const groupfoldersForGrafana = promRules[GRAFANA_RULES_SOURCE_NAME];

  //const [evaluateEveryChanged, setEvaluateEveryChanged]= useState(false)

  const someRulesBelongToThisGroup = (
    currentRule: string,
    promRules: RuleNamespace[],
    group: string,
    folder_: string
  ) => {
    const folderObj = promRules.find((ruleNameSpace) => ruleNameSpace.name === folder_);
    const groupObj = folderObj?.groups.find((ruleGroup) => ruleGroup.name === group);
    const rules = groupObj?.rules ?? [];
    return rules.find((rule) => rule.name !== currentRule);
  };

  const onBlur = () => setEditInterval(false);
  const evaluateEveryValidationOptions: RegisterOptions = {
    required: {
      value: true,
      message: 'Required.',
    },
    pattern: positiveDurationValidationPattern,
    validate: (value: string) => {
      const duration = parseDuration(value);
      if (Object.keys(duration).length) {
        const diff = durationToMilliseconds(duration);
        if (diff < MIN_TIME_RANGE_STEP_S * 1000) {
          return `Cannot be less than ${MIN_TIME_RANGE_STEP_S} seconds.`;
        }
        if (diff % (MIN_TIME_RANGE_STEP_S * 1000) !== 0) {
          return `Must be a multiple of ${MIN_TIME_RANGE_STEP_S} seconds.`;
        }
      }
      return true;
    },
  };

  useEffect(() => {
    group &&
      folder &&
      setValue(
        'evaluateEvery',
        secondsToHms(getIntervalForGroup(groupfoldersForGrafana?.result ?? [], group, folder?.title ?? ''))
      );
  }, [group, folder, groupfoldersForGrafana?.result, setValue]);

  // useEffect(() => setEvaluateEveryChanged(false), [folder, group])

  useEffect(() => {
    editInterval && setFocus('evaluateEvery');
  }, [editInterval, setFocus]);

  const intervalHasChanged =
    secondsToHms(getIntervalForGroup(groupfoldersForGrafana?.result ?? [], group, folder?.title ?? '')) !==
    evaluateEvery;

  return (
    <div>
      <FolderAndGroup initialFolder={initialFolder} />
      <Card className={styles.cardContainer}>
        <Card.Heading>Group behaviour</Card.Heading>
        <Card.Meta>
          {`Evaluation interval applies to every rule within a group. 
            It can overwrite the interval of an existing alert rule. 
            Click on 'Edit group behaviour' button to edit this group value.`}
        </Card.Meta>
        <Card.Actions>
          {intervalHasChanged &&
          !editInterval &&
          someRulesBelongToThisGroup(alertName, groupfoldersForGrafana?.result ?? [], group, folder?.title ?? '') ? (
            <>
              <div className={styles.intervalChangedLabel}>
                <Icon name="exclamation-triangle" size="xs" className={cx('text-warning', styles.warningIcon)} />
                {`More alert rules belong to this group.You are going to update evaluation interval for group '${group}' with this new value: `}
                <span className={cx('text-warning')}>{evaluateEvery}</span>
              </div>
              <div className={cx('text-warning')}>
                You should review the For duration, for all alerts that belong to this group
              </div>
            </>
          ) : (
            intervalHasChanged &&
            !editInterval && (
              <div className={styles.intervalChangedLabel}>
                {`Evaluation interval for group '${group}' will be updated with: `}
                <span className={cx('text-warning')}>{evaluateEvery}</span>
              </div>
            )
          )}

          {!editInterval && (
            <Button
              icon={editInterval ? 'exclamation-circle' : 'edit'}
              type="button"
              variant="secondary"
              onClick={() => {
                setEditInterval(true);
              }}
            >
              <span>{'Edit group behaviour'}</span>
            </Button>
          )}

          {editInterval && (
            <div className={styles.flexRow}>
              <div className={styles.evaluateLabel}>{`Alert rules in '${group}' are evaluated every`}</div>
              <Field
                className={styles.inlineField}
                error={errors.evaluateEvery?.message}
                invalid={!!errors.evaluateEvery}
                validationMessageHorizontalOverflow={true}
              >
                <Input
                  id={evaluateEveryId}
                  width={8}
                  {...register('evaluateEvery', evaluateEveryValidationOptions)}
                  readOnly={!editInterval}
                  onBlur={onBlur}
                  className={styles.evaluateInput}
                />
              </Field>

              <span className={cx('text-warning', styles.evalEditingLabel)}>
                {`You are updating evaluation interval for the group '${group}'`}
              </span>
            </div>
          )}
        </Card.Actions>
      </Card>
    </div>
  );
}

function ForInput() {
  const styles = useStyles2(getStyles);
  const {
    register,
    formState: { errors },
    watch,
  } = useFormContext<RuleFormValues>();

  const evaluateForId = 'eval-for-input';

  return (
    <div className={styles.flexRow}>
      <InlineLabel
        htmlFor={evaluateForId}
        width={7}
        tooltip='Once condition is breached, alert will go into pending state. If it is pending for longer than the "for" value, it will become a firing alert.'
      >
        for
      </InlineLabel>
      <Field
        className={styles.inlineField}
        error={errors.evaluateFor?.message}
        invalid={!!errors.evaluateFor?.message}
        validationMessageHorizontalOverflow={true}
      >
        <Input
          id={evaluateForId}
          width={8}
          {...register('evaluateFor', forValidationOptions(watch('evaluateEvery')))}
        />
      </Field>
    </div>
  );
}

export function GrafanaEvaluationBehavior({ initialFolder }: { initialFolder: RuleForm | null }) {
  const styles = useStyles2(getStyles);
  const [showErrorHandling, setShowErrorHandling] = useState(false);
  const { watch } = useFormContext<RuleFormValues>();

  const { exceedsLimit: exceedsGlobalEvaluationLimit } = checkEvaluationIntervalGlobalLimit(watch('evaluateEvery'));

  return (
    // TODO remove "and alert condition" for recording rules
    <RuleEditorSection stepNo={2} title="Alert evaluation behavior">
      <div className={styles.flexColumn}>
        <EvaluationIntervalInput initialFolder={initialFolder} />
        <ForInput />
      </div>
      {exceedsGlobalEvaluationLimit && <EvaluationIntervalLimitExceeded />}
      <CollapseToggle
        isCollapsed={!showErrorHandling}
        onToggle={(collapsed) => setShowErrorHandling(!collapsed)}
        text="Configure no data and error handling"
        className={styles.collapseToggle}
      />
      {showErrorHandling && (
        <>
          <Field htmlFor="no-data-state-input" label="Alert state if no data or all values are null">
            <InputControl
              render={({ field: { onChange, ref, ...field } }) => (
                <GrafanaAlertStatePicker
                  {...field}
                  inputId="no-data-state-input"
                  width={42}
                  includeNoData={true}
                  includeError={false}
                  onChange={(value) => onChange(value?.value)}
                />
              )}
              name="noDataState"
            />
          </Field>
          <Field htmlFor="exec-err-state-input" label="Alert state if execution error or timeout">
            <InputControl
              render={({ field: { onChange, ref, ...field } }) => (
                <GrafanaAlertStatePicker
                  {...field}
                  inputId="exec-err-state-input"
                  width={42}
                  includeNoData={false}
                  includeError={true}
                  onChange={(value) => onChange(value?.value)}
                />
              )}
              name="execErrState"
            />
          </Field>
        </>
      )}
    </RuleEditorSection>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  inlineField: css`
    margin-bottom: 0;
  `,
  flexColumn: css`
    display: flex;
    flex-direction: column;
    justify-content: flex-start;
    align-items: flex-start;
  `,
  flexRow: css`
    display: flex;
    flex-direction: row;
    justify-content: flex-start;
    align-items: flex-start;
  `,
  collapseToggle: css`
    margin: ${theme.spacing(2, 0, 2, -1)};
  `,
  globalLimitValue: css`
    font-weight: ${theme.typography.fontWeightBold};
  `,
  evaluateLabel: css`
    align-self: center;
    margin-right: ${theme.spacing(1)};
  `,
  evaluateInput: css`
    margin-right: ${theme.spacing(1)};
  `,
  alignBaseline: css`
    align-items: baseline;
    margin-bottom: ${theme.spacing(1)};
  `,
  formInput: css`
    width: 275px;

    & + & {
      margin-left: ${theme.spacing(3)};
    }
  `,
  evalEditingLabel: css`
    align-self: baseline;
    margin-left: ${theme.spacing(1)};
  `,
  cardContainer: css`
    max-width: ${theme.breakpoints.values.sm}px;
  `,
  intervalChangedLabel: css`
    margin-bottom: ${theme.spacing(1)};
  `,
  warningIcon: css`
    justify-self: center;
    margin-right: ${theme.spacing(1)};
  `,
});
