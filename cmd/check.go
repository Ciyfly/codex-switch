package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/codex-switch/codex-switch/internal/api"
	"github.com/codex-switch/codex-switch/internal/config"
	"github.com/codex-switch/codex-switch/internal/display"
	"github.com/codex-switch/codex-switch/internal/logging"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	checkAll       bool
	checkRefresh   bool
	checkParallel  int
	checkSilentBar bool
)

func init() {
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "检查额度使用情况",
		RunE:  runCheck,
	}

	checkCmd.Flags().BoolVar(&checkAll, "all", false, "检查所有 Key")
	checkCmd.Flags().BoolVar(&checkRefresh, "refresh", false, "调用远端 API 刷新额度")
	checkCmd.Flags().IntVar(&checkParallel, "parallel", 4, "并发请求数量")
	checkCmd.Flags().BoolVar(&checkSilentBar, "silent", false, "静默模式，不显示进度条")

	RootCommand().AddCommand(checkCmd)
}

type checkResult struct {
	id    string
	usage *api.UsageResult
	err   error
}

func runCheck(cmd *cobra.Command, _ []string) error {
	manager, err := mustLoadManager(cmd)
	if err != nil {
		return err
	}

	cfg, err := manager.Config()
	if err != nil {
		return err
	}

	var targets []config.APIKey
	if checkAll {
		targets = cfg.Keys
	} else {
		key, err := manager.ActiveKey()
		if err != nil {
			return err
		}
		targets = []config.APIKey{key}
	}

	if len(targets) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "没有可检查的 Key")
		return nil
	}

	logging.Infof("开始额度检查，目标数量: %d，刷新: %v", len(targets), checkRefresh)

	results := make([]checkResult, len(targets))
	bar := &progressbar.ProgressBar{}
	if !checkSilentBar {
		bar = progressbar.NewOptions(len(targets),
			progressbar.OptionSetDescription("正在检查额度..."),
			progressbar.OptionShowBytes(false),
			progressbar.OptionSetElapsedTime(false),
			progressbar.OptionSetPredictTime(false),
		)
	}

	var wg sync.WaitGroup
	semSize := checkParallel
	if semSize <= 0 {
		semSize = 1
	}
	sem := make(chan struct{}, semSize)

	for i, key := range targets {
		wg.Add(1)
		i := i
		key := key
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res := checkResult{id: key.ID}
			if checkRefresh {
				client, err := api.NewClient(key)
				if err != nil {
					res.err = err
				} else {
					usage, err := client.FetchUsage(key)
					if err != nil {
						res.err = err
					} else {
						res.usage = usage
						_ = manager.UpdateUsage(key.ID, usage.Used, usage.Limit, time.Now().UTC())
						logging.Debugf("刷新额度成功: %s", key.Name)
					}
				}
			}

			results[i] = res
			if !checkSilentBar {
				_ = bar.Add(1)
			}
		}()
	}

	wg.Wait()
	if !checkSilentBar {
		_ = bar.Finish()
		fmt.Fprintln(cmd.OutOrStdout())
	}

	if checkRefresh {
		if err := manager.Save(); err != nil {
			return err
		}
		cfg, err = manager.Config()
		if err != nil {
			return err
		}
	}

	keyMap := make(map[string]config.APIKey)
	for _, k := range cfg.Keys {
		keyMap[k.ID] = k
	}

	var failed []error
	totalUsed := 0.0
	for _, res := range results {
		key := keyMap[res.id]
		if checkRefresh && res.usage == nil && res.err == nil {
			// 如果未刷新成功，提供当前使用数据
			res.usage = &api.UsageResult{Used: key.QuotaUsed, Limit: key.QuotaLimit, Monthly: key.QuotaUsed}
		}
		display.PrintUsageReport(cmd.OutOrStdout(), key, res.usage)
		fmt.Fprintln(cmd.OutOrStdout())
		if res.err != nil {
			failed = append(failed, res.err)
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠ 查询 %s 失败: %v\n", key.Name, res.err)
			logging.Warnf("额度查询失败: %s (%s): %v", key.Name, key.ID, res.err)
		}
		totalUsed += key.QuotaUsed
	}

	fmt.Fprintf(cmd.OutOrStdout(), "合计已使用额度: $%.2f\n", totalUsed)
	if len(failed) > 0 {
		logging.Warnf("额度检查完成，%d 条失败", len(failed))
		return fmt.Errorf("%d 个查询失败，请查看上方警告", len(failed))
	}
	logging.Infof("额度检查完成，总使用 %.2f", totalUsed)
	return nil
}
