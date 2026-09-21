import { untrack } from 'solid-js';
import { createStore } from 'solid-js/store';

import {
    upstreamRuleGroupDelete,
    upstreamRuleGroupRefresh,
    upstreamRuleGroupSave,
    upstreamRuleGroupsStatus,
    upstreamRuleTest,
} from 'panel/api/generated';
import intl from 'panel/common/intl';
import { addErrorToast, addSuccessToast } from 'panel/stores/toasts';

import type { UpstreamRuleGroup } from 'panel/api/model/upstreamRuleGroup';
import type { UpstreamRuleIssue } from 'panel/api/model/upstreamRuleIssue';
import type { UpstreamRuleTestResponse } from 'panel/api/model/upstreamRuleTestResponse';

type UpstreamRulesState = {
    processing: boolean;
    processingAction: boolean;
    processingTest: boolean;
    defaultUpstreams: string[];
    groups: UpstreamRuleGroup[];
    issues: UpstreamRuleIssue[];
    testResult?: UpstreamRuleTestResponse;
};

const initialState: UpstreamRulesState = {
    processing: false,
    processingAction: false,
    processingTest: false,
    defaultUpstreams: [],
    groups: [],
    issues: [],
};

const [state, setState] = createStore<UpstreamRulesState>(initialState);

export const getUpstreamRuleGroups = async () => {
    setState('processing', true);
    try {
        const data = await upstreamRuleGroupsStatus();
        setState({
            processing: false,
            defaultUpstreams: data.default_upstreams || [],
            groups: data.groups || [],
            issues: data.issues || [],
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processing', false);
    }
};

export const saveUpstreamRuleGroup = async (group: UpstreamRuleGroup): Promise<boolean> => {
    setState('processingAction', true);
    try {
        await upstreamRuleGroupSave(group);
        addSuccessToast(intl.getMessage('changes_saved_success'));
        await getUpstreamRuleGroups();
        setState('processingAction', false);

        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingAction', false);

        return false;
    }
};

export const deleteUpstreamRuleGroup = async (id: number): Promise<boolean> => {
    setState('processingAction', true);
    try {
        await upstreamRuleGroupDelete({ id });
        addSuccessToast(intl.getMessage('changes_saved_success'));
        await getUpstreamRuleGroups();
        setState('processingAction', false);

        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingAction', false);

        return false;
    }
};

export const refreshUpstreamRuleGroup = async (id: number): Promise<boolean> => {
    setState('processingAction', true);
    try {
        await upstreamRuleGroupRefresh({ id });
        addSuccessToast(intl.getMessage('changes_saved_success'));
        await getUpstreamRuleGroups();
        setState('processingAction', false);

        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingAction', false);

        return false;
    }
};

export const refreshAllUpstreamRuleGroups = async (): Promise<boolean> => {
    setState('processingAction', true);
    try {
        for (const group of state.groups) {
            if (group.kind === 'subscription' && group.id !== undefined) {
                await upstreamRuleGroupRefresh({ id: group.id });
            }
        }
        addSuccessToast(intl.getMessage('changes_saved_success'));
        await getUpstreamRuleGroups();
        setState('processingAction', false);

        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingAction', false);

        return false;
    }
};

export const testUpstreamRule = async (domain: string) => {
    setState('processingTest', true);
    try {
        const testResult = await upstreamRuleTest({ domain });
        setState({ testResult, processingTest: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processingTest', false);
    }
};

export const upstreamRulesState = untrack(() => state);
