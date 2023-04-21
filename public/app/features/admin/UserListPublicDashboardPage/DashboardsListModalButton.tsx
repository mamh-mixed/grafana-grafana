import { css, cx } from '@emotion/css';
import React from 'react';

import { GrafanaTheme2 } from '@grafana/data/src';
import { Button, LinkButton, Modal, ModalsController, useStyles2 } from '@grafana/ui/src';
import {
  generatePublicDashboardUrl,
  SessionPublicDashboard,
} from 'app/features/dashboard/components/ShareModal/SharePublicDashboard/SharePublicDashboardUtils';

const DashboardsListModal = ({
  dashboards,
  onDismiss,
}: {
  dashboards: SessionPublicDashboard[];
  onDismiss: () => void;
}) => {
  const styles = useStyles2(getStyles);

  return (
    <Modal className={styles.modal} isOpen title="Public Dashboards" onDismiss={onDismiss}>
      {dashboards.map((dash) => (
        <div key={dash.name} className={styles.listItem}>
          <p className={styles.dashboardTitle}>{dash.name}</p>
          <div className={styles.urlsContainer}>
            <a
              className={styles.url}
              href={generatePublicDashboardUrl(dash.publicDashboardAccessToken)}
              onClick={onDismiss}
            >
              Public dashboard URL
            </a>
            <span className={styles.urlsDivider}>•</span>
            <a className={styles.url} href={`/d/${dash.dashboardUid}?shareView=share`} onClick={onDismiss}>
              Public dashboard settings
            </a>
          </div>
          <hr className={styles.divider} />
        </div>
      ))}
    </Modal>
  );
};

export const DashboardsListModalButton = ({ dashboards }: { dashboards: SessionPublicDashboard[] }) => (
  <ModalsController>
    {({ showModal, hideModal }) => (
      <Button
        variant="secondary"
        size="sm"
        icon="question-circle"
        title="Open dashboards list"
        onClick={() => showModal(DashboardsListModal, { dashboards, onDismiss: hideModal })}
        data-testid="query-tab-help-button"
      />
    )}
  </ModalsController>
);

const getStyles = (theme: GrafanaTheme2) => ({
  modal: css`
    width: 590px;
  `,
  listItem: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(0.5)};
  `,
  divider: css`
    margin: ${theme.spacing(1.5, 0)};
    color: ${theme.colors.text.secondary};
  `,
  urlsContainer: css`
    display: flex;
    gap: ${theme.spacing(0.5)};

    ${theme.breakpoints.down('sm')} {
      flex-direction: column;
    }
  `,
  urlsDivider: css`
    color: ${theme.colors.text.secondary};
    ${theme.breakpoints.down('sm')} {
      display: none;
    }
  `,
  dashboardTitle: css`
    font-size: ${theme.typography.body.fontSize};
    font-weight: ${theme.typography.fontWeightBold};
    margin-bottom: 0;
  `,
  url: css`
    font-size: ${theme.typography.body.fontSize};
    color: ${theme.colors.primary.text};
    text-decoration: underline;
  `,
});
